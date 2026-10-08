// SPDX-License-Identifier: Apache-2.0
import { useId, type ReactNode } from 'react';
import {
  Button,
  Card,
  Field,
  FormControls,
  EmptyState,
  StatusBadge,
} from '../../../brand/kit/web/react/server';
export { Button, Card, EmptyState, StatusBadge };
export function Input({
  label,
  value,
  onChange,
  type = 'text',
  description,
  required = false,
}: {
  label: string;
  value: string | number;
  onChange: (value: string) => void;
  type?: string;
  description?: string;
  required?: boolean;
}) {
  const id = useId();
  return (
    <Field
      id={id}
      label={label}
      description={description}
      control={
        <FormControls.TextInput
          type={type}
          value={value}
          onChange={(e) => onChange(e.target.value)}
          required={required}
          autoComplete={type === 'password' ? 'new-password' : 'off'}
        />
      }
    />
  );
}
export function Select({
  label,
  value,
  onChange,
  options,
}: {
  label: string;
  value: string;
  onChange: (value: string) => void;
  options: (string | { value: string; label: string })[];
}) {
  const id = useId();
  return (
    <Field
      id={id}
      label={label}
      control={
        <FormControls.Select
          value={value}
          onChange={(e) => onChange(e.target.value)}
        >
          {options.map((o) => (
            <option
              key={typeof o === 'string' ? o : o.value}
              value={typeof o === 'string' ? o : o.value}
            >
              {typeof o === 'string' ? o : o.label}
            </option>
          ))}
        </FormControls.Select>
      }
    />
  );
}
export function Area({
  label,
  value,
  onChange,
  description,
}: {
  label: string;
  value: string;
  onChange: (v: string) => void;
  description?: string;
}) {
  const id = useId();
  return (
    <Field
      id={id}
      label={label}
      description={description}
      control={
        <FormControls.Textarea
          value={value}
          rows={4}
          onChange={(e) => onChange(e.target.value)}
        />
      }
    />
  );
}
export function Check({
  label,
  value,
  onChange,
}: {
  label: string;
  value: boolean;
  onChange: (v: boolean) => void;
}) {
  const id = useId();
  return (
    <Field
      id={id}
      label={label}
      layout="inline"
      control={
        <FormControls.Checkbox
          checked={value}
          onChange={(e) => onChange(e.target.checked)}
        />
      }
    />
  );
}
export function Actions({ children }: { children: ReactNode }) {
  return <div className="actions">{children}</div>;
}
export function Facts({ value }: { value: unknown }) {
  if (value === null || value === undefined) return <span>Unknown</span>;
  if (
    typeof value === 'number' &&
    !Number.isSafeInteger(value) &&
    Number.isInteger(value)
  )
    return <span>Exact integer available in captured bytes</span>;
  if (typeof value !== 'object') return <span>{String(value)}</span>;
  if (Array.isArray(value))
    return (
      <ul>
        {value.map((v, i) => (
          <li key={i}>
            <Facts value={v} />
          </li>
        ))}
      </ul>
    );
  return (
    <dl className="facts">
      {Object.entries(value).map(([key, v]) => (
        <div key={key}>
          <dt>{key.replaceAll('_', ' ')}</dt>
          <dd>
            <Facts value={v} />
          </dd>
        </div>
      ))}
    </dl>
  );
}
export function Table({
  caption,
  heads,
  children,
}: {
  caption: string;
  heads: string[];
  children: ReactNode;
}) {
  return (
    <div className="table-scroll">
      <table data-bb-density="compact">
        <caption>{caption}</caption>
        <thead>
          <tr>
            {heads.map((h) => (
              <th key={h} scope="col">
                {h}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>{children}</tbody>
      </table>
    </div>
  );
}
