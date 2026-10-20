export function basePath() {
  return new URL(
    document.querySelector('base')?.href || '/',
    window.location.origin,
  ).pathname.replace(/\/$/, '')
}
