export function Textarea({ id, label, required = false, error, ...props }) {
  return (
    <div className="in-field">
      <label className="in-field__label" htmlFor={id}>{label}{required ? <span className="in-field__required" aria-hidden="true"> *</span> : null}</label>
      <textarea className="in-field__control" id={id} required={required} {...props} />
      {error ? <p className="in-field__error" role="alert">{error}</p> : null}
    </div>
  );
}
