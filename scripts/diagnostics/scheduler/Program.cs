// SPDX-License-Identifier: Apache-2.0
using System.Text;
using System.Text.Json;
using Microsoft.Windows.EventTracing;
using Microsoft.Windows.EventTracing.Cpu;
using Microsoft.Windows.EventTracing.Processes;
using SchedulerState = Microsoft.Windows.EventTracing.Cpu.ThreadState;

record Owned(int pid, long query_begin_unix_micros, long query_end_unix_micros, long created_unix_micros);
record Anchor(long unix_micros, double begin_elapsed_seconds, double end_elapsed_seconds);
record Input(long begin_unix_micros, long end_unix_micros, Owned[] owned, int gui_pid, int wrapper_pid, Anchor start_anchor, Anchor end_anchor);
record Span(string kind, long begin, long end, int? reason = null);
record Ready(long time, bool resolved, bool resident);
sealed class ProjectionFailure(string code) : Exception { internal string Code { get; } = code; }

static class Projection
{
    // Fixed conservative clock tolerance, not a PID-reuse exemption. The trace
    // process-start timestamp and GetProcessTimes creation clock are distinct.
    internal const long ClockToleranceUs = 20_000;
    internal static bool Lifetime(Owned o, long start, long? stop) =>
        Math.Abs(start - o.created_unix_micros) <= ClockToleranceUs &&
        start <= o.query_begin_unix_micros && (!stop.HasValue || stop.Value >= o.query_end_unix_micros);

    internal static void Alignment(Input input)
    {
        var a = input.start_anchor; var b = input.end_anchor;
        if (a == null || b == null || a.unix_micros <= 0 || b.unix_micros <= a.unix_micros ||
            !double.IsFinite(a.begin_elapsed_seconds) || !double.IsFinite(a.end_elapsed_seconds) ||
            !double.IsFinite(b.begin_elapsed_seconds) || !double.IsFinite(b.end_elapsed_seconds) ||
            a.end_elapsed_seconds < a.begin_elapsed_seconds || b.end_elapsed_seconds < b.begin_elapsed_seconds ||
            a.end_elapsed_seconds - a.begin_elapsed_seconds > .020 || b.end_elapsed_seconds - b.begin_elapsed_seconds > .020 ||
            b.begin_elapsed_seconds <= a.end_elapsed_seconds ||
            Math.Abs((b.unix_micros - a.unix_micros) -
                ((b.begin_elapsed_seconds + b.end_elapsed_seconds - a.begin_elapsed_seconds - a.end_elapsed_seconds) * 500_000)) > ClockToleranceUs ||
            input.begin_unix_micros < a.unix_micros || input.end_unix_micros > b.unix_micros)
            throw new ProjectionFailure("Unavailable clock alignment");
    }
    internal static long Clip(long a, long b, long begin, long end) => Math.Max(0, Math.Min(b, end) - Math.Max(a, begin));

    internal static Span[] Tail(bool occupied, SchedulerState? state, Ready[] ready, long begin, long end, int? reason)
    {
        if (occupied || ready.Any(r => !r.resolved || !r.resident || r.time < begin || r.time >= end)) return Array.Empty<Span>();
        if (state == SchedulerState.Ready) return new[] { new Span("ready", begin, end) };
        if (state != SchedulerState.Waiting) return Array.Empty<Span>();
        long split = ready.Select(r => r.time).DefaultIfEmpty(end).Min();
        return split == end ? new[] { new Span("waiting", begin, end, reason) } :
            new[] { new Span("waiting", begin, split, reason), new Span("ready", split, end) };
    }

    internal static object Summary(IEnumerable<Span> spans, long begin, long end)
    {
        var raw = spans.ToArray();
        if (end <= begin || raw.Any(s => s.end < s.begin || !new[] { "scheduled", "ready", "waiting" }.Contains(s.kind))) throw new ProjectionFailure("Invalid span");
        var selected = raw.Select(s => new Span(s.kind, Math.Max(begin, s.begin), Math.Min(end, s.end), s.reason))
            .Where(s => s.end > s.begin).OrderBy(s => s.begin).ThenBy(s => s.end).ToList();
        long covered = 0, previous = begin;
        foreach (var s in selected)
        {
            if (s.begin < previous || !new[] { "scheduled", "ready", "waiting" }.Contains(s.kind))
                throw new ProjectionFailure("Invalid interval projection");
            covered += s.end - s.begin; previous = s.end;
        }
        object Kind(string kind)
        {
            var rows = selected.Where(s => s.kind == kind).ToList();
            return new { intervals = rows.Count, total_ns = rows.Sum(s => s.end - s.begin), max_ns = rows.Select(s => s.end - s.begin).DefaultIfEmpty(0).Max() };
        }
        return new { observed_ns = end - begin, covered_ns = covered, unknown_ns = end - begin - covered,
            scheduled = Kind("scheduled"), ready = Kind("ready"), waiting = Kind("waiting"),
            wait_reasons = selected.Where(s => s.kind == "waiting" && s.reason.HasValue).GroupBy(s => s.reason!.Value)
                .OrderBy(g => g.Key).Select(g => new { reason = g.Key, intervals = g.Count(), total_ns = g.Sum(s => s.end - s.begin) }).ToArray() };
    }

    internal static string Role(string image, string command, int pid, int gui, int wrapper)
    {
        if (pid == gui) return "gui";
        if (pid == wrapper) return "wrapper";
        if (!string.Equals(Path.GetFileName(image), "msedgewebview2.exe", StringComparison.OrdinalIgnoreCase)) return "other-owned";
        foreach (var pair in new[] { ("renderer", "renderer"), ("gpu-process", "gpu"), ("utility", "utility"), ("crashpad-handler", "crashpad") })
            if (System.Text.RegularExpressions.Regex.IsMatch(command, @"(?:^|[\s" + '"' + @"])--type=" + pair.Item1 + @"(?:$|[\s" + '"' + @"])", System.Text.RegularExpressions.RegexOptions.CultureInvariant)) return pair.Item2;
        return command.Contains("--type=", StringComparison.Ordinal) ? "other-owned" : "browser";
    }

    internal static void SelfTest()
    {
        if (Clip(0, 30, 10, 20) != 10 || Clip(0, 5, 10, 20) != 0) throw new Exception("Clipping regression");
        var encoded = JsonSerializer.Serialize(Summary(new[] { new Span("waiting", 0, 12, 13), new Span("ready", 12, 17), new Span("scheduled", 17, 25) }, 10, 20));
        using var result = JsonDocument.Parse(encoded);
        var r = result.RootElement;
        if (r.GetProperty("waiting").GetProperty("total_ns").GetInt64() != 2 || r.GetProperty("ready").GetProperty("total_ns").GetInt64() != 5 || r.GetProperty("scheduled").GetProperty("total_ns").GetInt64() != 3 || r.GetProperty("unknown_ns").GetInt64() != 0) throw new Exception("State accounting regression");
        using var censored = JsonDocument.Parse(JsonSerializer.Serialize(Summary(new[] { new Span("scheduled", 12, 14) }, 10, 20)));
        if (censored.RootElement.GetProperty("unknown_ns").GetInt64() != 8) throw new Exception("Unknown interval regression");
        bool rejected = false;
        try { Summary(new[] { new Span("ready", 0, 15), new Span("scheduled", 12, 20) }, 10, 20); } catch (ProjectionFailure) { rejected = true; }
        if (!rejected) throw new Exception("Overlapping interval regression");
        var owned = new Owned(5, 200_000, 210_000, 100_000);
        if (!Lifetime(owned, 100_100, 220_000) || Lifetime(owned, 100_100, 205_000) || Lifetime(owned, 199_000, 220_000) || Lifetime(owned, 10_000, 150_000)) throw new Exception("Process incarnation regression");
        var aligned = new Input(1_000_000, 2_000_000, new[] { owned }, 5, 6, new Anchor(1_000_000, 0, 0), new Anchor(2_000_000, 1, 1));
        Alignment(aligned);
        rejected = false;
        try { Alignment(aligned with { end_anchor = new Anchor(2_100_000, 1, 1) }); } catch (ProjectionFailure) { rejected = true; }
        if (!rejected) throw new Exception("Clock discontinuity regression");
        var tail = Tail(false, SchedulerState.Waiting, new[] { new Ready(17, true, true) }, 10, 20, 13);
        if (tail.Length != 2 || tail[0].kind != "waiting" || tail[0].end != 17 || tail[1].kind != "ready" || tail[1].begin != 17) throw new Exception("Censored tail split regression");
        if (Tail(false, SchedulerState.Waiting, Array.Empty<Ready>(), 10, 20, 13).Single().end != 20 ||
            Tail(false, SchedulerState.Ready, Array.Empty<Ready>(), 10, 20, null).Single().kind != "ready" ||
            Tail(true, SchedulerState.Waiting, Array.Empty<Ready>(), 10, 20, null).Length != 0 ||
            Tail(false, SchedulerState.Waiting, new[] { new Ready(17, false, true) }, 10, 20, null).Length != 0 ||
            Tail(false, SchedulerState.Waiting, new[] { new Ready(17, true, false) }, 10, 20, null).Length != 0)
            throw new Exception("Unknown tail regression");
        if (Role("msedgewebview2.exe", "--type=renderer", 3, 1, 2) != "renderer" || Role("other.exe", "--type=renderer", 3, 1, 2) != "other-owned" || Role("msedgewebview2.exe", "--type=renderer-extra", 3, 1, 2) != "other-owned") throw new Exception("Role projection regression");
    }
}

class Program
{
    static readonly JsonSerializerOptions Json = new() { WriteIndented = false };
    static string Stage = "validate-input";
    static readonly Dictionary<string, long> Evidence = new();
    static long Us(DateTimeOffset value) => (value.UtcTicks - DateTimeOffset.UnixEpoch.UtcTicks) / 10;
    static long Ns(TraceTimestamp value) => value.Nanoseconds;
    static bool Full(TraceTimestamp value) => value.HasValue && !value.IsPartial;
    static bool Full(TraceDuration? value) => value.HasValue && value.Value.HasValue && !value.Value.IsPartial && value.Value.Nanoseconds >= 0;

    static object Decode(string tracePath, Input input)
    {
        Projection.Alignment(input);
        if (input.owned.Length is < 1 or > 16 || input.begin_unix_micros <= 0 || input.end_unix_micros <= input.begin_unix_micros || input.end_unix_micros - input.begin_unix_micros > 20_000_000 || input.owned.Select(o => o.pid).Distinct().Count() != input.owned.Length)
            throw new ProjectionFailure("Invalid owned observation");
        Stage = "open-trace";
        using var trace = TraceProcessor.Create(tracePath, new TraceProcessorSettings { AllowLostEvents = false, AllowTimeInversion = false, SuppressFirstTimeSetupMessage = true });
        Stage = "configure-consumers";
        var pending = trace.UseCpuSchedulingData();
        var metadata = trace.UseMetadata();
        Stage = "process-trace";
        trace.Process(Stream.Null, Stream.Null);
        Stage = "validate-coverage";
        var cpu = pending.Result;
        Evidence["lost_events"] = metadata.LostEventCount; Evidence["lost_buffers"] = metadata.LostBufferCount;
        Evidence["context_switch_count"] = cpu.ContextSwitches.Count; Evidence["ready_event_count"] = cpu.ReadyThreadEvents.Count;
        if (metadata.LostEventCount != 0 || metadata.LostBufferCount != 0 || cpu.ContextSwitches.Count == 0 || cpu.ReadyThreadEvents.Count == 0) throw new ProjectionFailure("Missing complete scheduler trace");
        long traceStart = Us(metadata.StartTime);
        long begin = checked((input.begin_unix_micros - traceStart) * 1000), end = checked((input.end_unix_micros - traceStart) * 1000);
        var first = cpu.ContextSwitches.Min(c => Ns(c.Timestamp)); var last = cpu.ContextSwitches.Max(c => Ns(c.Timestamp));
        if (first > begin || last < end) {
            Evidence["trace_start_unix_micros"] = traceStart; Evidence["window_begin_ns"] = begin; Evidence["window_end_ns"] = end;
            Evidence["first_context_switch_ns"] = first; Evidence["last_context_switch_ns"] = last;
            throw new ProjectionFailure("Incomplete case coverage");
        }
        Evidence["trace_start_unix_micros"] = traceStart;
        Evidence["window_begin_ns"] = begin; Evidence["window_end_ns"] = end;
        Evidence["first_context_switch_ns"] = first; Evidence["last_context_switch_ns"] = last;
        var candidates = cpu.ThreadActivity.Select(a => a.Process).Where(p => p != null).Distinct().ToArray();
        var processes = new List<object>();
        foreach (var owned in input.owned)
        {
            Stage = "validate-owned-lifetime";
            if (owned.pid <= 0 || owned.created_unix_micros <= 0 || owned.query_end_unix_micros < owned.query_begin_unix_micros || owned.created_unix_micros > owned.query_begin_unix_micros ||
                owned.query_begin_unix_micros < input.begin_unix_micros || owned.query_end_unix_micros > input.end_unix_micros || owned.created_unix_micros < traceStart)
                throw new ProjectionFailure("Invalid ownership identity");
            var incarnations = candidates.Where(p => p!.Id == owned.pid).ToArray();
            var matches = incarnations.Where(p => p!.CreateTime.HasValue && Full(p.CreateTime.Value)
                && (!p.ExitTime.HasValue || Full(p.ExitTime.Value))
                && Projection.Lifetime(owned, Us(p.CreateTime.Value.DateTimeOffset), p.ExitTime.HasValue ? Us(p.ExitTime.Value.DateTimeOffset) : null)).ToArray();
            Evidence["incarnation_count"] = incarnations.Length; Evidence["matched_incarnation_count"] = matches.Length;
            Evidence["native_birth_unix_micros"] = owned.created_unix_micros;
            Evidence["nearest_birth_delta_us"] = incarnations.Where(p => p!.CreateTime.HasValue && Full(p.CreateTime.Value))
                .Select(p => Us(p!.CreateTime!.Value.DateTimeOffset) - owned.created_unix_micros).OrderBy(Math.Abs).DefaultIfEmpty(long.MaxValue).First();
            // A second incarnation touching the ownership bracket is ambiguous,
            // even if only one passes the stricter native-birth check.
            if (incarnations.Count(p => !p!.CreateTime.HasValue || !Full(p.CreateTime.Value) ||
                (Us(p.CreateTime.Value.DateTimeOffset) <= owned.query_end_unix_micros &&
                 (!p.ExitTime.HasValue || !Full(p.ExitTime.Value) || Us(p.ExitTime.Value.DateTimeOffset) >= owned.query_begin_unix_micros))) != 1)
                throw new ProjectionFailure("Conflicting process incarnation");
            if (matches.Length != 1) throw new ProjectionFailure("Ambiguous owned process lifetime");
            var process = matches[0]!;
            Stage = "project-owned-threads";
            var groups = cpu.ThreadActivity.Where(a => a.Process == process && a.Thread != null).GroupBy(a => a.Thread!).ToArray();
            if (groups.Length > 256) throw new ProjectionFailure("Thread bound exceeded");
            var threads = new List<(int id, long start, long maxReady, long maxWait, object row)>();
            foreach (var group in groups)
            {
                var thread = group.Key;
                Evidence["projection_tid"] = thread.Id;
                long lifeBegin = begin, lifeEnd = end;
                if (thread.CreateTime.HasValue && Full(thread.CreateTime.Value)) lifeBegin = Math.Max(lifeBegin, Ns(thread.CreateTime.Value));
                if (thread.ExitTime.HasValue && Full(thread.ExitTime.Value)) lifeEnd = Math.Min(lifeEnd, Ns(thread.ExitTime.Value));
                if (lifeEnd <= lifeBegin) continue;
                var spans = new List<Span>();
                foreach (var activity in group)
                {
                    if (!Full(activity.StartTime)) continue;
                    long start = Ns(activity.StartTime);
                    if (Full(activity.StopTime)) spans.Add(new Span("scheduled", start, Ns(activity.StopTime)));
                    if (Full(activity.ReadyDuration))
                    {
                        long ready = activity.ReadyDuration!.Value.Nanoseconds;
                        spans.Add(new Span("ready", start - ready, start));
                        if (Full(activity.WaitingDuration))
                            spans.Add(new Span("waiting", start - ready - activity.WaitingDuration!.Value.Nanoseconds, start - ready));
                    }
                }
                // A final off-CPU interval is not reported as zero. Only a known
                // switch-out state plus complete readiness coverage can fill it.
                var final = group.Where(a => Full(a.StopTime) && Ns(a.StopTime) < lifeEnd).OrderBy(a => Ns(a.StopTime)).LastOrDefault();
                if (final?.SwitchOut != null && !group.Any(a => Full(a.StartTime) && Ns(a.StartTime) >= Ns(final.StopTime) && Ns(a.StartTime) < lifeEnd))
                {
                    long tail = Math.Max(lifeBegin, Ns(final.StopTime));
                    var ready = cpu.ReadyThreadEvents.Where(r => r.ReadiedThreadId == thread.Id && Full(r.Timestamp) && Ns(r.Timestamp) >= tail && Ns(r.Timestamp) < lifeEnd).OrderBy(r => Ns(r.Timestamp)).ToArray();
                    // Swapped-out readiness can include transition time; leave
                    // this tail unknown rather than equating it with ready time.
                    spans.AddRange(Projection.Tail(spans.Any(s => s.end > tail && s.begin < lifeEnd), final.SwitchOut.ThreadState,
                        ready.Select(r => new Ready(Ns(r.Timestamp), r.ReadiedThread == thread, r.IsKernelStackResident && r.IsProcessResident)).ToArray(),
                        tail, lifeEnd, final.SwitchOut.WaitReason.HasValue ? (int)final.SwitchOut.WaitReason.Value : null));
                }
                var summary = Projection.Summary(spans, lifeBegin, lifeEnd);
                long maxReady = spans.Where(s => s.kind == "ready").Select(s => Projection.Clip(s.begin, s.end, lifeBegin, lifeEnd)).DefaultIfEmpty(0).Max();
                long maxWait = spans.Where(s => s.kind == "waiting").Select(s => Projection.Clip(s.begin, s.end, lifeBegin, lifeEnd)).DefaultIfEmpty(0).Max();
                long born = thread.CreateTime.HasValue && Full(thread.CreateTime.Value) ? Ns(thread.CreateTime.Value) : long.MaxValue;
                threads.Add((thread.Id, born, maxReady, maxWait, new { tid = thread.Id, observation_begin_ns = lifeBegin - begin, observation_end_ns = lifeEnd - begin,
                    birth_observed = thread.CreateTime.HasValue && Full(thread.CreateTime.Value), exit_observed = thread.ExitTime.HasValue && Full(thread.ExitTime.Value), wait_reason_coverage = "known-tail-switch-out-only", summary }));
            }
            var chosen = threads.OrderBy(t => t.start).Take(1).Concat(threads.OrderByDescending(t => t.maxReady).Take(4)).Concat(threads.OrderByDescending(t => t.maxWait).Take(4)).DistinctBy(t => (t.id, t.start)).ToArray();
            processes.Add(new { pid = owned.pid, role = Projection.Role(process.ImageName ?? "", process.CommandLine ?? "", owned.pid, input.gui_pid, input.wrapper_pid),
                role_source = "verified-owned-lifetime-and-fixed-command-switch", matched_lifetime = true,
                native_created_unix_micros = owned.created_unix_micros, trace_created_unix_micros = Us(process.CreateTime!.Value.DateTimeOffset),
                thread_count = threads.Count, selected_thread_count = chosen.Length, selection = "oldest-plus-four-longest-ready-and-four-longest-waiting",
                threads = chosen.Select(t => t.row).ToArray() });
        }
        return new { kind = "owned-scheduler-observation", available = true, feature_qualification = false, lost_events = metadata.LostEventCount,
            lost_buffers = metadata.LostBufferCount, coverage = "bounded-sequential-no-loss-context-switch-range-and-owned-lifetimes", clock_tolerance_us = Projection.ClockToleranceUs, window_begin_unix_micros = input.begin_unix_micros,
            window_end_unix_micros = input.end_unix_micros, scheduled_interpretation = "scheduled-not-exact-executed-cpu", processes };
    }

    static int Main(string[] args)
    {
        if (args.Length == 1 && args[0] == "--self-test") { Projection.SelfTest(); Console.WriteLine("{\"projection_self_test\":\"passed\",\"recording_started\":false}"); return 0; }
        if (args.Length != 3) return 78;
        object result; int code = 0;
        try
        {
            if (new FileInfo(args[0]).Length >= 256L * 1024 * 1024 || new FileInfo(args[1]).Length > 65536) throw new ProjectionFailure("Input bound exceeded");
            var input = JsonSerializer.Deserialize<Input>(File.ReadAllText(args[1])) ?? throw new ProjectionFailure("Missing input");
            result = Decode(args[0], input);
        }
        catch (Exception error)
        {
            code = 78;
            result = new { kind = "owned-scheduler-observation", available = false, feature_qualification = false,
                reason = "trace-decoder-unavailable", stage = Stage, validation = error is ProjectionFailure rejection ? rejection.Code : null,
                hresult = error.HResult, inner_hresult = error.InnerException?.HResult, counters = Evidence };
        }
        var encoded = JsonSerializer.Serialize(result, Json) + "\n";
        if (Encoding.UTF8.GetByteCount(encoded) > 65536) return 78;
        File.WriteAllText(args[2], encoded, new UTF8Encoding(false));
        return code;
    }
}
