import { cloneElement, type ButtonHTMLAttributes, type InputHTMLAttributes, type ReactElement, type ReactNode, type SelectHTMLAttributes, type TextareaHTMLAttributes } from "react";

type WithoutVisualEscape<T> = Omit<T, "className" | "style" | "color">;
type Variant = "primary" | "secondary" | "ghost" | "destructive";

export function AppFrame({ children, host = "browser", header, layout = "contained" }: { children: ReactNode; host?: "browser" | "tauri" | "wails"; header?: ReactNode; layout?: "contained" | "full-bleed" }) {
  return <div className="bb-app-frame" data-bb-app-frame data-bb-host={host} data-bb-layout={layout}><a className="bb-skip-link bb-control" href="#bb-main">Skip to content</a>{header}<div className="bb-app-frame__scroll"><main id="bb-main" className="bb-app-frame__content" tabIndex={-1}>{children}</main></div><div id="bb-overlay-root" className="bb-app-frame__overlay" data-bb-overlay-root /></div>;
}

export function Button({ variant = "primary", loading = false, children, disabled, ...props }: WithoutVisualEscape<ButtonHTMLAttributes<HTMLButtonElement>> & { variant?: Variant; loading?: boolean }) {
  return <button type="button" className={`bb-control bb-button bb-button--${variant}`} aria-busy={loading || undefined} disabled={disabled || loading} {...props}>{loading ? <span role="status">Loading</span> : children}</button>;
}

export function IconButton({ label, icon, variant = "ghost", ...props }: WithoutVisualEscape<Omit<ButtonHTMLAttributes<HTMLButtonElement>, "aria-label">> & { label: string; icon: ReactNode; variant?: Variant }) {
  return <button type="button" className={`bb-control bb-icon-button bb-button--${variant}`} {...props} aria-label={label}><span aria-hidden="true">{icon}</span></button>;
}

export function Field({ id, label, description, error, layout = "stacked", control }: { id: string; label: string; description?: string; error?: string; layout?: "stacked" | "inline"; control: ReactElement<{ id?: string; "aria-describedby"?: string; "aria-invalid"?: boolean }> }) {
  const describedBy = [control.props["aria-describedby"], description ? `${id}-description` : null, error ? `${id}-error` : null].filter(Boolean).join(" ") || undefined;
  const boundControl = cloneElement(control, { id, "aria-describedby": describedBy, "aria-invalid": error ? true : control.props["aria-invalid"] });
  return <div className={`bb-field bb-field--${layout}`}><label htmlFor={id}>{label}</label>{description ? <span id={`${id}-description`}>{description}</span> : null}{boundControl}{error ? <span id={`${id}-error`} role="alert">{error}</span> : null}</div>;
}

export function TextInput(props: WithoutVisualEscape<InputHTMLAttributes<HTMLInputElement>>) { return <input className="bb-field__control" {...props} />; }
export function Textarea(props: WithoutVisualEscape<TextareaHTMLAttributes<HTMLTextAreaElement>>) { return <textarea className="bb-field__control" {...props} />; }
export function Select(props: WithoutVisualEscape<SelectHTMLAttributes<HTMLSelectElement>>) { return <select className="bb-field__control" {...props} />; }
export function Checkbox(props: WithoutVisualEscape<Omit<InputHTMLAttributes<HTMLInputElement>, "type">>) { return <input type="checkbox" className="bb-field__control" {...props} />; }
export function Radio(props: WithoutVisualEscape<Omit<InputHTMLAttributes<HTMLInputElement>, "type">>) { return <input type="radio" className="bb-field__control" {...props} />; }
export const FormControls = { TextInput, Textarea, Select, Checkbox, Radio };

type ListRowProps =
  | { variant?: "static"; label: string; description?: string }
  | { variant: "action"; label: string; description?: string; href: string }
  | { variant: "action"; label: string; description?: string; onActivate: () => void }
  | { variant: "selection"; label: string; description?: string; selected: boolean; onSelect: () => void };
export function ListRow(props: ListRowProps) { const content = <><span>{props.label}</span>{props.description ? <span>{props.description}</span> : null}</>; if (props.variant === "action" && "href" in props) return <li className="bb-list-row"><a className="bb-list-row__action bb-control" href={props.href}>{content}</a></li>; if (props.variant === "action") return <li className="bb-list-row"><button type="button" className="bb-list-row__action bb-control" onClick={props.onActivate}>{content}</button></li>; if (props.variant === "selection") return <li className="bb-list-row"><button type="button" className="bb-list-row__action bb-control" aria-pressed={props.selected} onClick={props.onSelect}>{content}</button></li>; return <li className="bb-list-row">{content}</li>; }
export function SplitPaneFrame({ primary, separator, secondary }: { primary: ReactNode; separator: ReactNode; secondary: ReactNode }) { return <div className="bb-split-pane"><section>{primary}</section>{separator}<section>{secondary}</section></div>; }
export function StatusBadge({ status, children }: { status: "neutral" | "success" | "warning" | "error"; children: ReactNode }) { return <span className="bb-status-badge" data-status={status}>{children}</span>; }
export function Card({ heading, children, variant = "plain" }: { heading?: string; children: ReactNode; variant?: "plain" | "outlined" | "elevated" }) { return <article className={`bb-card bb-card--${variant}`}>{heading ? <h2>{heading}</h2> : null}{children}</article>; }
export function EmptyState({ id, heading, description, action, variant = "informational" }: { id: string; heading: string; description: string; action?: ReactNode; variant?: "informational" | "actionable" }) { return <section className="bb-empty-state" data-variant={variant} aria-labelledby={`${id}-heading`}><h2 id={`${id}-heading`}>{heading}</h2><p>{description}</p>{action}</section>; }
