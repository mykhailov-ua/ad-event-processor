export function cn(...parts: Array<string | false | null | undefined>): string {
  let out = '';
  for (let i = 0; i < parts.length; i += 1) {
    const part = parts[i];
    if (part) {
      out = out ? `${out} ${part}` : part;
    }
  }
  return out;
}
