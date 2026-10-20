// SPDX-License-Identifier: Apache-2.0
using System.Reflection;
using Microsoft.Windows.EventTracing;
using Microsoft.Windows.EventTracing.Cpu;
using Microsoft.Windows.EventTracing.Processes;
using SchedulerState = Microsoft.Windows.EventTracing.Cpu.ThreadState;

// Only --self-test uses these fixed interfaces and actual pinned activity
// constructors. No trace, native session or product process is created.
public class SchedulerPropertyFixture : DispatchProxy
{
    internal Dictionary<string, object?> Values = new();
    protected override object? Invoke(MethodInfo? method, object?[]? args) =>
        Values.TryGetValue(method!.Name, out var value) ? value : throw new NotSupportedException(method.Name);
    internal static T Make<T>(Dictionary<string, object?> values) where T : class
    {
        var value = Create<T, SchedulerPropertyFixture>();
        ((SchedulerPropertyFixture)(object)value).Values = values;
        return value;
    }
}

static class SchedulerFixtures
{
    internal static void Run()
    {
        var partial = typeof(TraceTimestamp).Assembly.GetType("Microsoft.Windows.EventTracing.PartialTraceTimestampContext", true)!;
        var context = Activator.CreateInstance(partial, BindingFlags.Instance | BindingFlags.Public | BindingFlags.NonPublic,
            null, new object[] { new ClockFixture() }, null)!;
        TraceTimestamp At(long ns) => (TraceTimestamp)partial.GetMethod("Create", new[] { typeof(Timestamp) })!.Invoke(context, new object[] { new Timestamp(ns) })!;
        var process = SchedulerPropertyFixture.Make<IProcess>(new() { ["get_Id"] = 55 });
        IThread Thread() => SchedulerPropertyFixture.Make<IThread>(new() { ["get_Id"] = 10, ["get_CreateTime"] = (TraceTimestamp?)At(100), ["get_Process"] = process });
        var thread = Thread();
        IContextSwitch Switch(long time, bool present = true) => SchedulerPropertyFixture.Make<IContextSwitch>(new() { ["get_Timestamp"] = present ? At(time) : default(TraceTimestamp) });
        IContextSwitchIn In(long time, IThread owner, bool present) => SchedulerPropertyFixture.Make<IContextSwitchIn>(new()
        { ["get_ContextSwitch"] = Switch(time, present), ["get_Thread"] = owner, ["get_ThreadId"] = 10, ["get_Process"] = process });
        IContextSwitchOut Out(long time, SchedulerState? state, IThread owner, bool present) => SchedulerPropertyFixture.Make<IContextSwitchOut>(new()
        { ["get_ContextSwitch"] = Switch(time, present), ["get_Thread"] = owner, ["get_ThreadId"] = 10, ["get_Process"] = process,
            ["get_ThreadState"] = state, ["get_WaitReason"] = (KernelWaitReason?)(KernelWaitReason)13 });
        IReadyThreadEvent Ready(long time, IThread? target, bool stack = true, bool resident = true, bool timestamp = true) =>
            SchedulerPropertyFixture.Make<IReadyThreadEvent2>(new()
            { ["get_Timestamp"] = timestamp ? At(time) : default(TraceTimestamp), ["get_ReadiedThreadId"] = 10,
                ["get_ReadiedThread"] = target, ["get_IsKernelStackResident"] = stack, ["get_IsProcessResident"] = resident });
        var concrete = typeof(ICpuThreadActivity).Assembly.GetType("Microsoft.Windows.EventTracing.Cpu.CpuThreadActivity", true)!;
        ICpuThreadActivity Activity(long begin, long end, SchedulerState? state = SchedulerState.Waiting,
            ICpuThreadActivity? previous = null, IReadyThreadEvent? ready = null, IThread? owner = null,
            bool startPresent = true, bool stopPresent = true, IThread? outOwner = null) =>
            (ICpuThreadActivity)Activator.CreateInstance(concrete, BindingFlags.Instance | BindingFlags.Public | BindingFlags.NonPublic,
                null, new object?[] { In(begin, owner ?? thread, startPresent), Out(end, state, outOwner ?? owner ?? thread, stopPresent), previous, null, ready, null, null }, null)!;
        ClosedProjection Project(ICpuThreadActivity activity, params IReadyThreadEvent[] events) =>
            Program.Closed(activity, thread, process, new ReadyIndex(events));
        void Require(bool condition, string message) { if (!condition) throw new Exception(message); }

        var initial = Activity(1000, 1100);
        Require(initial.ReadyDuration?.Nanoseconds == 900 && Project(initial).spans.Length == 0,
            "Pinned birth fallback must remain unknown");
        var missingReady = Activity(2000, 2100, previous: initial);
        Require(missingReady.ReadyDuration?.Nanoseconds == 900 && Project(missingReady).spans.Length == 0,
            "Pinned previous-Waiting fallback must remain unknown");
        var r = Ready(700, thread);
        var initialReady = Project(Activity(1000, 1100, ready: r), r);
        Require(initialReady.spans.Length == 1 && initialReady.spans[0].begin == 700 && initialReady.spans[0].end == 1000,
            "Initial explicit readiness must not infer waiting from birth");
        r = Ready(1700, thread);
        var split = Project(Activity(2000, 2100, previous: initial, ready: r), r);
        Require(split.spans.Single(s => s.kind == "waiting") == new Span("waiting", 1100, 1700, 13, "waiting-switch-out-to-ready-event") &&
            split.spans.Single(s => s.kind == "ready").begin == 1700, "Explicit waiting/readiness split regression");
        foreach (var invalid in new[] { Ready(1700, Thread()), Ready(1700, null), Ready(1700, thread, stack: false),
            Ready(1700, thread, resident: false), Ready(1700, thread, timestamp: false), Ready(2200, thread) })
            Require(Project(Activity(2000, 2100, previous: initial, ready: invalid), invalid).spans.Length == 0,
                "Incompatible readiness must remain unknown");
        var preempted = Activity(1000, 1100, SchedulerState.Ready);
        Require(Project(Activity(2000, 2100, previous: preempted)).spans.Single().begin == 1100,
            "Explicit Ready switch-out regression");
        Require(Project(Activity(2000, 2100, previous: preempted), Ready(1500, null)).spans.Length == 0,
            "Intervening unresolved same-TID event must block fallback");
        Require(Project(Activity(2000, 2100, previous: Activity(1000, 1100, SchedulerState.Ready, owner: Thread()))).spans.Length == 0,
            "Reused TID must not establish previous activity identity");
        Require(Project(Activity(2000, 2100, previous: Activity(1000, 1100, SchedulerState.Ready, outOwner: Thread()))).spans.Length == 0,
            "Switch-out identity must independently match");
        Require(Project(Activity(2000, 2100, previous: preempted, startPresent: false)).spans.Length == 0 &&
            Project(Activity(2000, 2100, previous: Activity(1000, 1100, SchedulerState.Ready, stopPresent: false))).spans.Length == 0,
            "Missing endpoints must not imply ready time");
        Require(Project(Activity(2000, 2100, previous: preempted), Ready(0, thread, timestamp: false)).spans.Length == 0,
            "Unknown same-TID event timestamps must block state inference");
        foreach (var state in new SchedulerState?[] { null, SchedulerState.Transition, SchedulerState.DeferredReady, SchedulerState.Terminated })
            Require(Project(Activity(2000, 2100, previous: Activity(1000, 1100, state))).spans.Length == 0,
                "Unsupported previous state must remain unknown");
        r = Ready(1098, thread);
        var crossing = Project(Activity(2000, 2100, previous: initial, ready: r), r);
        Require(crossing.spans.Length == 1 && crossing.spans[0].begin == 1098 && crossing.spans[0].kind == "ready",
            "Crossing readiness must not create negative waiting or snap its endpoint");
        var ambiguous = Projection.Unambiguous(crossing.spans.Append(new Span("scheduled", 1000, 1100)), 1000, 2100);
        Require(ambiguous.conflict_unknown_ns == 2, "Crossing activity adapter must preserve conflict uncertainty");
        r = Ready(1700, thread);
        Require(Project(Activity(2000, 2100, previous: initial, ready: r), r, Ready(1800, thread, resident: false)).spans.Length == 0,
            "Later nonresident event must invalidate ready classification");
        var earlier = Project(Activity(2000, 2100, previous: initial, ready: r), Ready(1400, thread), r);
        Require(earlier.spans.Length == 1 && earlier.spans[0].kind == "ready", "Earlier readiness must prevent fabricated waiting");
        r = Ready(2000, thread);
        var atSwitch = Activity(2000, 2100, previous: initial, ready: r);
        Require(Project(atSwitch).spans.Length == 0 && Project(atSwitch, Ready(2000, Thread())).spans.Length == 0,
            "Zero-length ready boundary still requires an exact raw witness");
        Require(Project(atSwitch, r).spans.Single(s => s.kind == "waiting").end == 2000,
            "Validated readiness at switch-in must preserve preceding waiting");
        r = Ready(1500, thread);
        var tail = Program.FinalTail(initial, thread, process, new ReadyIndex(new[] { r }), 3000, false);
        var clippedTail = Projection.Unambiguous(tail, 2000, 3000);
        Require(clippedTail.spans.Length == 1 && clippedTail.spans[0].kind == "ready" &&
            clippedTail.spans[0].begin == 2000 && clippedTail.spans[0].end == 3000,
            "Readiness before observation must not become waiting after clipping");
        Require(Program.FinalTail(Activity(1000, 1100, outOwner: Thread()), thread, process, new ReadyIndex(Array.Empty<IReadyThreadEvent>()), 3000, false).Length == 0,
            "Tail switch-out must use exact thread identity, not reused TID");
    }
}
