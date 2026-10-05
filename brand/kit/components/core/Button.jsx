export function Button({ variant = "primary", size = "md", children, ...props }) {
  return <button className={`in-button in-button--${variant} in-button--${size}`} type="button" {...props}>{children}</button>;
}
