export function Card({ children, ...props }) {
  return <div className="in-card" {...props}>{children}</div>;
}
