export function SectionHeading({ eyebrow, title, description }) {
  return <header className="in-section-heading">{eyebrow ? <div className="in-eyebrow">{eyebrow}</div> : null}<h2 className="in-section-heading__title">{title}</h2>{description ? <p className="in-section-heading__description">{description}</p> : null}</header>;
}
