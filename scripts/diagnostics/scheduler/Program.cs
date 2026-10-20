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
record Partition(Span[] spans, long conflict_unknown_ns, int conflict_intervals, object? first_conflict);
record Ready(long time, bool resolved, bool resident);
sealed class ProjectionFailure(string code, object? witness = null) : Exception
{
    internal string Code { get; } = code;
    internal object? Witness { get; } = witness;
}
sealed class ClockFixture : ITraceTimestampContext
{
    public TraceClockType ClockType => (TraceClockType)0;
    public TraceTimestampValue ReferenceValue => default;
    public DateTimeOffset ReferenceTime => DateTimeOffset.UnixEpoch;
    public bool IsPartial => false;
    public DateTimeOffset GetDateTimeOffset(TraceTimestampValue value) => throw new NotSupportedException();
    public Timestamp GetRelativeTimestamp(TraceTimestampValue value) => throw new NotSupportedException();
    public TraceTimestampValue GetApproximateValue(Timestamp value) => throw new NotSupportedException();
}

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

    // ReadyThread can precede the preceding switch-out (E018). Neither
    // category owns that conflicting portion. Keep it unknown, without a
    // tolerance or precedence rule; other overlapping states still reject.
    internal static Partition Unambiguous(IEnumerable<Span> source, long begin, long end)
    {
        var raw = source.ToArray();
        if (end <= begin || raw.Any(s => s.end < s.begin || !new[] { "scheduled", "ready", "waiting" }.Contains(s.kind)))
            throw new ProjectionFailure("Invalid span");
        var spans = raw.Select(s => s with { begin = Math.Max(begin, s.begin), end = Math.Min(end, s.end) })
            .Where(s => s.end > s.begin).ToArray();
        var events = spans.SelectMany((s, i) => new[] { (time: s.begin, id: i, add: true), (time: s.end, id: i, add: false) })
            .GroupBy(e => e.time).OrderBy(g => g.Key).ToArray();
        var active = new SortedSet<int>(); var known = new List<Span>();
        long conflict = 0; int conflicts = 0, previousId = -1; object? witness = null;
        for (int i = 0; i < events.Length - 1; i++)
        {
            foreach (var e in events[i]) { if (e.add) active.Add(e.id); else active.Remove(e.id); }
            long left = events[i].Key, right = events[i + 1].Key;
            if (active.Count == 1)
            {
                int id = active.Min;
                if (previousId == id && known.Count > 0 && known[^1].end == left)
                    known[^1] = known[^1] with { end = right };
                else known.Add(spans[id] with { begin = left, end = right });
                previousId = id;
            }
            else
            {
                previousId = -1;
                if (active.Count == 0) continue;
                var pair = active.Select(id => spans[id]).ToArray();
                if (pair.Length != 2 || !pair.Any(s => s.kind == "scheduled") || !pair.Any(s => s.kind == "ready"))
                    throw new ProjectionFailure("Invalid interval projection", new { previous = pair[0], current = pair[1],
                        overlap_ns = right - left, window_begin_ns = begin, window_end_ns = end });
                conflict += right - left; conflicts++;
                witness ??= new { previous = pair[0], current = pair[1], overlap_ns = right - left,
                    begin_ns = left, end_ns = right };
            }
        }
        return new Partition(known.ToArray(), conflict, conflicts, witness);
    }

    internal static object Summary(IEnumerable<Span> spans, long begin, long end)
    {
        var raw = spans.ToArray();
        if (end <= begin || raw.Any(s => s.end < s.begin || !new[] { "scheduled", "ready", "waiting" }.Contains(s.kind))) throw new ProjectionFailure("Invalid span");
        var selected = raw.Select(s => new Span(s.kind, Math.Max(begin, s.begin), Math.Min(end, s.end), s.reason))
            .Where(s => s.end > s.begin).OrderBy(s => s.begin).ThenBy(s => s.end).ToList();
        long covered = 0, previous = begin; Span? previousSpan = null;
        foreach (var s in selected)
        {
            if (s.begin < previous || !new[] { "scheduled", "ready", "waiting" }.Contains(s.kind))
                throw new ProjectionFailure("Invalid interval projection", new { previous = previousSpan, current = s,
                    overlap_ns = Math.Max(0, Math.Min(previous, s.end) - s.begin), window_begin_ns = begin, window_end_ns = end });
            covered += s.end - s.begin; previous = s.end; previousSpan = s;
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
        try { Summary(new[] { new Span("ready", 0, 15), new Span("scheduled", 12, 20) }, 10, 20); }
        catch (ProjectionFailure error)
        {
            using var witness = JsonDocument.Parse(JsonSerializer.Serialize(error.Witness));
            rejected = witness.RootElement.GetProperty("overlap_ns").GetInt64() == 3 &&
                witness.RootElement.GetProperty("previous").GetProperty("kind").GetString() == "ready" &&
                witness.RootElement.GetProperty("current").GetProperty("kind").GetString() == "scheduled";
        }
        if (!rejected) throw new Exception("Overlapping interval regression");
        // Exact E018 witness: discard only the 200ns disputed portion from
        // both categories, never choose ready or scheduled as the winner.
        var conflict = Unambiguous(new[] { new Span("scheduled", 362484300, 378039500),
            new Span("ready", 378039300, 378042300) }, 362484300, 378042300);
        using var conflictSummary = JsonDocument.Parse(JsonSerializer.Serialize(Summary(conflict.spans, 362484300, 378042300)));
        var cr = conflictSummary.RootElement;
        if (conflict.conflict_unknown_ns != 200 || conflict.conflict_intervals != 1 || conflict.first_conflict == null ||
            cr.GetProperty("unknown_ns").GetInt64() != 200 || cr.GetProperty("ready").GetProperty("total_ns").GetInt64() != 2800 ||
            cr.GetProperty("scheduled").GetProperty("total_ns").GetInt64() != 15555000)
            throw new Exception("E018 conflict uncertainty regression");
        var largeConflict = Unambiguous(new[] { new Span("scheduled", 0, 100), new Span("ready", 0, 100) }, 0, 100);
        if (largeConflict.spans.Length != 0 || largeConflict.conflict_unknown_ns != 100)
            throw new Exception("Large conflict must stay entirely unknown");
        var boundaryConflict = Unambiguous(new[] { new Span("scheduled", 0, 20), new Span("ready", 12, 30) }, 10, 25);
        if (boundaryConflict.conflict_unknown_ns != 8 || boundaryConflict.spans.Sum(s => s.end - s.begin) != 7)
            throw new Exception("Conflict clipping regression");
        var containedConflict = Unambiguous(new[] { new Span("scheduled", 0, 30), new Span("ready", 10, 20) }, 0, 30);
        if (containedConflict.conflict_unknown_ns != 10 || containedConflict.spans.Length != 2 ||
            containedConflict.spans.Max(s => s.end - s.begin) != 10)
            throw new Exception("Unknown must split contiguous maxima");
        var touching = Unambiguous(new[] { new Span("scheduled", 0, 10), new Span("ready", 10, 20) }, 0, 20);
        if (touching.conflict_unknown_ns != 0 || touching.spans.Length != 2)
            throw new Exception("Touching boundaries are not overlap");
        var beforeWindow = Unambiguous(new[] { new Span("scheduled", 0, 10), new Span("ready", 5, 20) }, 10, 20);
        var afterWindow = Unambiguous(new[] { new Span("scheduled", 0, 15), new Span("ready", 10, 20) }, 0, 10);
        if (beforeWindow.conflict_unknown_ns != 0 || afterWindow.conflict_unknown_ns != 0)
            throw new Exception("Outside-window conflicts must not contaminate known time");
        foreach (var invalid in new[] {
            new[] { new Span("scheduled", 0, 20), new Span("scheduled", 12, 30) },
            new[] { new Span("waiting", 0, 20), new Span("ready", 12, 30) },
            new[] { new Span("scheduled", 0, 20), new Span("ready", 12, 30), new Span("ready", 15, 25) } })
        {
            rejected = false;
            try { Unambiguous(invalid, 0, 30); } catch (ProjectionFailure) { rejected = true; }
            if (!rejected) throw new Exception("Structural overlap must remain rejected");
        }
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
    static readonly Dictionary<string, long?> Evidence = new();
    static long Us(DateTimeOffset value) => (value.UtcTicks - DateTimeOffset.UnixEpoch.UtcTicks) / 10;
    static long Ns(TraceTimestamp value) => value.Nanoseconds;
    static bool Known(TraceTimestamp value) => value.HasValue;
    static bool Known(TraceDuration? value) => value.HasValue && value.Value.HasValue && value.Value.Nanoseconds >= 0;

    static void ClockProjectionTest()
    {
        // Exercise the actual pinned adapter used by legacy process/thread data.
        // Its partial flag concerns raw clock Value, not interval coverage.
        var type = typeof(TraceTimestamp).Assembly.GetType("Microsoft.Windows.EventTracing.PartialTraceTimestampContext", true)!;
        var context = Activator.CreateInstance(type, System.Reflection.BindingFlags.Instance | System.Reflection.BindingFlags.Public | System.Reflection.BindingFlags.NonPublic,
            null, new object[] { new ClockFixture() }, null)!;
        var timestamp = (TraceTimestamp)type.GetMethod("Create", new[] { typeof(Timestamp) })!.Invoke(context, new object[] { new Timestamp(123_456_700) })!;
        var duration = (TraceDuration)type.GetMethod("Create", new[] { typeof(Duration) })!.Invoke(context, new object[] { new Duration(100_000_000) })!;
        if (!timestamp.IsPartial || !duration.IsPartial || timestamp.Nanoseconds != 123_456_700 || duration.Nanoseconds != 100_000_000 ||
            timestamp.DateTimeOffset != DateTimeOffset.UnixEpoch.AddTicks(1_234_567)) throw new Exception("Pinned timestamp projection regression");
        if (!Known(timestamp) || !Known(duration) || Known(default(TraceTimestamp)) || Known((TraceDuration?)null))
            throw new Exception("Usable normalized clock projection rejected");
        bool rejected = false;
        try { _ = timestamp.Value; } catch (InvalidOperationException) { rejected = true; }
        if (!rejected) throw new Exception("Raw clock value must remain unavailable");
    }

    static object Decode(string tracePath, Input input)
    {
        Projection.Alignment(input);
        if (input.owned.Length is < 1 or > 16 || input.begin_unix_micros <= 0 || input.end_unix_micros <= input.begin_unix_micros || input.end_unix_micros - input.begin_unix_micros > 20_000_000 || input.owned.Select(o => o.pid).Distinct().Count() != input.owned.Length)
            throw new ProjectionFailure("Invalid owned observation");
        Stage = "open-trace";
        using var trace = TraceProcessor.Create(tracePath, new TraceProcessorSettings { AllowLostEvents = false, AllowTimeInversion = false, SuppressFirstTimeSetupMessage = true });
        Stage = "configure-consumers";
        var pending = trace.UseCpuSchedulingData();
        var pendingProcesses = trace.UseProcesses();
        trace.UseThreads();
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
        var candidates = pendingProcesses.Result.Processes.ToArray();
        var processes = new List<object>();
        foreach (var owned in input.owned)
        {
            Stage = "validate-owned-lifetime";
            if (owned.pid <= 0 || owned.created_unix_micros <= 0 || owned.query_end_unix_micros < owned.query_begin_unix_micros || owned.created_unix_micros > owned.query_begin_unix_micros ||
                owned.query_begin_unix_micros < input.begin_unix_micros || owned.query_end_unix_micros > input.end_unix_micros || owned.created_unix_micros < traceStart)
                throw new ProjectionFailure("Invalid ownership identity");
            var incarnations = candidates.Where(p => p!.Id == owned.pid).ToArray();
            var matches = incarnations.Where(p => p!.CreateTime.HasValue && Known(p.CreateTime.Value)
                && (!p.ExitTime.HasValue || Known(p.ExitTime.Value))
                && Projection.Lifetime(owned, Us(p.CreateTime.Value.DateTimeOffset), p.ExitTime.HasValue ? Us(p.ExitTime.Value.DateTimeOffset) : null)).ToArray();
            Evidence["incarnation_count"] = incarnations.Length; Evidence["matched_incarnation_count"] = matches.Length;
            Evidence["usable_birth_count"] = incarnations.Count(p => p!.CreateTime.HasValue && Known(p.CreateTime.Value));
            Evidence["raw_clock_birth_count"] = incarnations.Count(p => p!.CreateTime.HasValue && Known(p.CreateTime.Value) && !p.CreateTime.Value.IsPartial);
            Evidence["normalized_birth_count"] = incarnations.Count(p => p!.CreateTime.HasValue && p.CreateTime.Value.HasValue && p.CreateTime.Value.IsPartial);
            Evidence["missing_birth_count"] = incarnations.Count(p => !p!.CreateTime.HasValue || !p.CreateTime.Value.HasValue);
            Evidence["native_birth_unix_micros"] = owned.created_unix_micros;
            Evidence["nearest_birth_delta_us"] = incarnations.Where(p => p!.CreateTime.HasValue && Known(p.CreateTime.Value))
                .Select(p => (long?)(Us(p!.CreateTime!.Value.DateTimeOffset) - owned.created_unix_micros)).OrderBy(value => Math.Abs(value!.Value)).FirstOrDefault();
            // A second incarnation touching the ownership bracket is ambiguous,
            // even if only one passes the stricter native-birth check.
            if (incarnations.Count(p => !p!.CreateTime.HasValue || !Known(p.CreateTime.Value) ||
                (Us(p.CreateTime.Value.DateTimeOffset) <= owned.query_end_unix_micros &&
                 (!p.ExitTime.HasValue || !Known(p.ExitTime.Value) || Us(p.ExitTime.Value.DateTimeOffset) >= owned.query_begin_unix_micros))) != 1)
                throw new ProjectionFailure("Conflicting process incarnation");
            if (matches.Length != 1) throw new ProjectionFailure("Ambiguous owned process lifetime");
            var process = matches[0]!;
            Stage = "project-owned-threads";
            Evidence["projection_pid"] = owned.pid;
            Evidence["same_pid_activity_count"] = cpu.ThreadActivity.Count(a => a.Process?.Id == owned.pid);
            Evidence["matched_process_activity_count"] = cpu.ThreadActivity.Count(a => a.Process == process);
            var groups = cpu.ThreadActivity.Where(a => a.Process == process && a.Thread != null).GroupBy(a => a.Thread!).ToArray();
            if (groups.Length > 256) throw new ProjectionFailure("Thread bound exceeded");
            var threads = new List<(int id, long start, long maxReady, long maxWait, object row)>();
            foreach (var group in groups)
            {
                var thread = group.Key;
                Evidence["projection_tid"] = thread.Id;
                long lifeBegin = begin, lifeEnd = end;
                if (thread.CreateTime.HasValue && Known(thread.CreateTime.Value)) lifeBegin = Math.Max(lifeBegin, Ns(thread.CreateTime.Value));
                if (thread.ExitTime.HasValue && Known(thread.ExitTime.Value)) lifeEnd = Math.Min(lifeEnd, Ns(thread.ExitTime.Value));
                if (lifeEnd <= lifeBegin) continue;
                var spans = new List<Span>();
                foreach (var activity in group)
                {
                    if (!Known(activity.StartTime)) continue;
                    long start = Ns(activity.StartTime);
                    if (Known(activity.StopTime)) spans.Add(new Span("scheduled", start, Ns(activity.StopTime)));
                    if (Known(activity.ReadyDuration))
                    {
                        long ready = activity.ReadyDuration!.Value.Nanoseconds;
                        spans.Add(new Span("ready", start - ready, start));
                        if (Known(activity.WaitingDuration))
                            spans.Add(new Span("waiting", start - ready - activity.WaitingDuration!.Value.Nanoseconds, start - ready));
                    }
                }
                // A final off-CPU interval is not reported as zero. Only a known
                // switch-out state plus complete readiness coverage can fill it.
                var final = group.Where(a => Known(a.StopTime) && Ns(a.StopTime) < lifeEnd).OrderBy(a => Ns(a.StopTime)).LastOrDefault();
                if (final?.SwitchOut != null && !group.Any(a => Known(a.StartTime) && Ns(a.StartTime) >= Ns(final.StopTime) && Ns(a.StartTime) < lifeEnd))
                {
                    long tail = Math.Max(lifeBegin, Ns(final.StopTime));
                    var ready = cpu.ReadyThreadEvents.Where(r => r.ReadiedThreadId == thread.Id && Known(r.Timestamp) && Ns(r.Timestamp) >= tail && Ns(r.Timestamp) < lifeEnd).OrderBy(r => Ns(r.Timestamp)).ToArray();
                    // Swapped-out readiness can include transition time; leave
                    // this tail unknown rather than equating it with ready time.
                    spans.AddRange(Projection.Tail(spans.Any(s => s.end > tail && s.begin < lifeEnd), final.SwitchOut.ThreadState,
                        ready.Select(r => new Ready(Ns(r.Timestamp), r.ReadiedThread == thread, r.IsKernelStackResident && r.IsProcessResident)).ToArray(),
                        tail, lifeEnd, final.SwitchOut.WaitReason.HasValue ? (int)final.SwitchOut.WaitReason.Value : null));
                }
                var partition = Projection.Unambiguous(spans, lifeBegin, lifeEnd);
                var summary = Projection.Summary(partition.spans, lifeBegin, lifeEnd);
                long maxReady = partition.spans.Where(s => s.kind == "ready").Select(s => s.end - s.begin).DefaultIfEmpty(0).Max();
                long maxWait = partition.spans.Where(s => s.kind == "waiting").Select(s => s.end - s.begin).DefaultIfEmpty(0).Max();
                long born = thread.CreateTime.HasValue && Known(thread.CreateTime.Value) ? Ns(thread.CreateTime.Value) : long.MaxValue;
                threads.Add((thread.Id, born, maxReady, maxWait, new { tid = thread.Id, observation_begin_ns = lifeBegin - begin, observation_end_ns = lifeEnd - begin,
                    birth_observed = thread.CreateTime.HasValue && Known(thread.CreateTime.Value), exit_observed = thread.ExitTime.HasValue && Known(thread.ExitTime.Value),
                    conflict_unknown_ns = partition.conflict_unknown_ns, conflict_intervals = partition.conflict_intervals, first_conflict = partition.first_conflict,
                    wait_reason_coverage = "known-tail-switch-out-only", summary }));
            }
            var chosen = threads.OrderBy(t => t.start).Take(1).Concat(threads.OrderByDescending(t => t.maxReady).Take(4)).Concat(threads.OrderByDescending(t => t.maxWait).Take(4)).DistinctBy(t => (t.id, t.start)).ToArray();
            processes.Add(new { pid = owned.pid, role = Projection.Role(process.ImageName ?? "", process.CommandLine ?? "", owned.pid, input.gui_pid, input.wrapper_pid),
                role_source = "verified-owned-lifetime-and-fixed-command-switch", matched_lifetime = true,
                native_created_unix_micros = owned.created_unix_micros, trace_created_unix_micros = Us(process.CreateTime!.Value.DateTimeOffset),
                thread_count = threads.Count, selected_thread_count = chosen.Length, selection = "oldest-plus-four-longest-ready-and-four-longest-waiting",
                threads = chosen.Select(t => t.row).ToArray() });
        }
        return new { kind = "owned-scheduler-observation", available = true, feature_qualification = false, lost_events = metadata.LostEventCount,
            lost_buffers = metadata.LostBufferCount, coverage = "bounded-sequential-no-loss-context-switch-range-and-owned-lifetimes", time_projection = "supported-relative-and-clock-time-without-raw-clock-value", clock_tolerance_us = Projection.ClockToleranceUs, window_begin_unix_micros = input.begin_unix_micros,
            window_end_unix_micros = input.end_unix_micros, scheduled_interpretation = "unambiguous-scheduled-lower-bound-not-exact-executed-cpu",
            overlap_policy = "scheduled-ready-conflicts-unknown-without-tolerance-other-overlaps-reject",
            state_accounting = "known-fragment-totals-and-contiguous-maxima-lower-bounds-unknown-explicit", counters = Evidence, processes };
    }

    static int Main(string[] args)
    {
        if (args.Length == 1 && args[0] == "--self-test") { Projection.SelfTest(); ClockProjectionTest(); Console.WriteLine("{\"projection_self_test\":\"passed\",\"recording_started\":false}"); return 0; }
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
                hresult = error.HResult, inner_hresult = error.InnerException?.HResult, counters = Evidence,
                witness = error is ProjectionFailure projection ? projection.Witness : null };
        }
        var encoded = JsonSerializer.Serialize(result, Json) + "\n";
        if (Encoding.UTF8.GetByteCount(encoded) > 65536) return 78;
        File.WriteAllText(args[2], encoded, new UTF8Encoding(false));
        return code;
    }
}
