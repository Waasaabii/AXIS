export function compilePattern(pattern) {
  if (!pattern) {
    return null;
  }

  let source = pattern;
  let flags = "";

  if (source.startsWith("(?i)")) {
    source = source.slice(4);
    flags += "i";
  }

  return new RegExp(source, flags);
}
