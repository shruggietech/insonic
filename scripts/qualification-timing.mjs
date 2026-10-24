// SPDX-License-Identifier: Apache-2.0
// Maintainer test allowances, deliberately separate from product deadlines.
import committed from '../internal/qualificationtiming/policy.json' with { type: 'json' };

export function validateTimingPolicy(value) {
  if (!value || typeof value !== 'object' || Array.isArray(value) || !Object.keys(value).length
      || Object.entries(value).some(([key, seconds]) => !/^[a-z][a-z_]*_seconds$/.test(key)
        || !Number.isSafeInteger(seconds) || seconds < 1 || seconds * 1000 > 2 ** 31 - 1)) {
    throw new Error('Invalid qualification timing policy.');
  }
  return Object.freeze({ ...value });
}

export const qualificationTiming = validateTimingPolicy(committed);

export function milliseconds(name) {
  const seconds = qualificationTiming[name];
  if (!Number.isSafeInteger(seconds)) throw new Error('Missing qualification timing allowance: ' + name);
  return seconds * 1000;
}

export function validateAggregatePolicy(value) {
  const keys = ['qualification_target_seconds', 'qualification_limit_seconds', 'finalization_reserve_seconds'];
  if (!value || typeof value !== 'object' || Array.isArray(value) || Object.keys(value).length !== keys.length
      || keys.some(key => !Number.isSafeInteger(value[key]) || value[key] < 1 || value[key] * 1000 > 2 ** 31 - 1)
      || value.qualification_target_seconds > value.qualification_limit_seconds
      || value.finalization_reserve_seconds >= value.qualification_limit_seconds) {
    throw new Error('Invalid aggregate qualification timing policy.');
  }
  return Object.freeze({ ...value });
}

// An enclosing phase retains its original absolute origin across nested waits.
export function effectiveDeadline(now, allowanceMs, phaseDeadlineMs) {
  if (!Number.isFinite(now) || !Number.isSafeInteger(allowanceMs) || allowanceMs < 1
      || (phaseDeadlineMs !== undefined && (!Number.isSafeInteger(phaseDeadlineMs) || phaseDeadlineMs < 1))) {
    throw new Error('Invalid qualification deadline.');
  }
  const deadline = Math.min(now + allowanceMs, phaseDeadlineMs ?? Infinity);
  if (!Number.isSafeInteger(deadline) || deadline <= now) throw new Error('Original qualification deadline exceeded.');
  return deadline;
}

export async function withinAllowance(operation, allowanceMs, message = 'Qualification operation deadline exceeded.') {
  if (!Number.isSafeInteger(allowanceMs) || allowanceMs < 1 || allowanceMs > 2 ** 31 - 1) throw new Error('Invalid qualification operation allowance.');
  const deadline = performance.now() + allowanceMs;
  let timer;
  try {
    const result = await Promise.race([Promise.resolve().then(operation), new Promise((_, reject) => {
      timer = setTimeout(() => reject(new Error(message)), allowanceMs);
    })]);
    if (performance.now() > deadline) throw new Error(message);
    return result;
  } finally { clearTimeout(timer); }
}
