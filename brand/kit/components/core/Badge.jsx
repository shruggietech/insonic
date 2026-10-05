export function Badge({ tone = "neutral", children }) {
  return <span className={`in-badge in-badge--${tone}`}>{children}</span>;
}
