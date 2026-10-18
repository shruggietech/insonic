# S037 presentation and lifecycle model

No persisted catalog or configuration entities change.

| Entity | State/identity | Rules |
| --- | --- | --- |
| Compact desktop host | existing Wails application root | delivered tokens, intrinsic actions, shrinking grid/flex children, contained table overflow |
| Selection control | native input and associated label | square visual glyph; whole label target at least44 by44; Space/click/native checked/disabled behavior |
| Exit controller | waiting / ready / requested / dispatched | queue before startup, dispatch at most once after a non-nil lifecycle context, do not own runtime cancellation |
| Native Exit evidence | host/source/scenario/session/work identifiers | no-workspace and open-workspace activation; external runtime/work remains alive; playback ownership closes |
| Density evidence | clean source, bundle/binary hashes, scenario geometry and images | renderer identity is explicit; OS/native claims require actual native observations |
