export function Select({ id, label, required = false, error, children, ...props }) {
  return (
    <div className="in-field">
      <label className="in-field__label" htmlFor={id}>{label}{required ? <span className="in-field__required" aria-hidden="true"> *</span> : null}</label>
      <select className="in-field__control" id={id} required={required} {...props}>{children}</select>
      {error ? <p className="in-field__error" role="alert">{error}</p> : null}
    </div>
  );
}
