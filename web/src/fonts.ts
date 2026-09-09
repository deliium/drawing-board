/** Optional DEV font load probe. Log prefix: [fonts] */

const isDev =
  typeof import.meta !== 'undefined' &&
  Boolean((import.meta as { env?: { DEV?: boolean } }).env?.DEV)

export function probeCriticalFonts(): void {
  if (!isDev || typeof document === 'undefined' || !document.fonts?.load) return
  void Promise.all([
    document.fonts.load('400 16px "IBM Plex Sans"'),
    document.fonts.load('400 16px "Noto Sans JP"'),
  ])
    .then(() => {
      console.debug('[fonts] faces registered')
    })
    .catch(() => {
      console.warn('[fonts] WARN critical face failed to load')
    })
}
