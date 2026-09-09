/** Client-side correction display by stable feedback code. Log: [corrections.i18n] */

import { getLocale, interpolate, type Locale } from './index'

export const CORRECTION_CODES = [
  'empty_strokes',
  'stroke_count_mismatch',
  'stroke_order',
  'start_direction',
  'end_direction',
  'relative_placement',
  'proportions',
  'shape',
] as const

export type CorrectionCode = (typeof CORRECTION_CODES)[number]

export type CorrectionParams = {
  glyph?: string
  want?: number
  got?: number
  strokeN?: number
}

const isDev =
  typeof import.meta !== 'undefined' &&
  Boolean((import.meta as { env?: { DEV?: boolean } }).env?.DEV)

function corrDebug(...args: unknown[]) {
  if (isDev) console.debug('[corrections.i18n]', ...args)
}

const enCatalog: Record<CorrectionCode, string> = {
  empty_strokes: 'No strokes were submitted. Draw the character, then try again.',
  stroke_count_mismatch:
    'Use {want} strokes for 「{glyph}」 (you used {got}). Assessed attempts are immutable — start a new attempt and focus on stroke count.',
  stroke_order:
    'Check stroke order — start with the stroke that begins at the top/left for 「{glyph}」. Start a new attempt after this one.',
  start_direction:
    'Start stroke {strokeN} in the same direction as the model (see the tip of the first movement). Retry on a new attempt.',
  end_direction: 'Finish stroke {strokeN} in the expected direction. Retry on a new attempt.',
  relative_placement:
    'Place the strokes closer to their usual positions relative to each other. Start a new attempt focusing on placement.',
  proportions:
    'Adjust the length or size of the strokes so parts of 「{glyph}」 match usual proportions. Retry on a new attempt.',
  shape:
    'The overall shape of 「{glyph}」 still differs from the model — slow down and retrace on a new attempt.',
}

const enCatalogNoStroke: Partial<Record<CorrectionCode, string>> = {
  start_direction: 'Start the stroke in the same direction as the model. Retry on a new attempt.',
  end_direction: 'Finish the stroke in the expected direction. Retry on a new attempt.',
}

const jaCatalog: Record<CorrectionCode, string> = {
  empty_strokes: '筆画がありません。書いてからもう一度試してください。',
  stroke_count_mismatch:
    '「{glyph}」は{want}画です（今は{got}画）。採点済みの挑戦は変更できません — 新しい挑戦で画数に集中してください。',
  stroke_order:
    '筆順を確認してください — 「{glyph}」は上／左から始まる画から。このあと新しい挑戦を始めてください。',
  start_direction:
    '{strokeN}画目の書き始めの方向を見本に合わせてください（最初の動きの先端を確認）。新しい挑戦でやり直してください。',
  end_direction: '{strokeN}画目の終わりの方向を見本に合わせてください。新しい挑戦でやり直してください。',
  relative_placement:
    '各画の位置関係を見本に近づけてください。配置に集中して新しい挑戦を始めてください。',
  proportions:
    '「{glyph}」の各部分の長さや大きさを見本の比率に合わせてください。新しい挑戦でやり直してください。',
  shape:
    '「{glyph}」全体の形がまだ見本と違います — ゆっくり新しい挑戦でなぞり直してください。',
}

const jaCatalogNoStroke: Partial<Record<CorrectionCode, string>> = {
  start_direction: '書き始めの方向を見本に合わせてください。新しい挑戦でやり直してください。',
  end_direction: '終わりの方向を見本に合わせてください。新しい挑戦でやり直してください。',
}

const fallbackUnknown: Record<Locale, string> = {
  en: 'Review the model character and try again on a new attempt.',
  ja: '見本の文字を確認し、新しい挑戦でもう一度試してください。',
}

function isCorrectionCode(code: string): code is CorrectionCode {
  return (CORRECTION_CODES as readonly string[]).includes(code)
}

/**
 * Map feedback code + params to localized display text.
 * Falls back to API `message` when code unknown or template missing.
 */
export function formatCorrectionDisplay(
  code: string,
  apiMessage: string,
  params?: CorrectionParams,
  locale: Locale = getLocale(),
): string {
  if (!isCorrectionCode(code)) {
    corrDebug('fallback api message', code)
    return apiMessage?.trim() || fallbackUnknown[locale]
  }

  // Prefer API message when parametric codes lack required context (MVP: API stores EN).
  if (code === 'stroke_count_mismatch' && (params?.want == null || params?.got == null || !params?.glyph)) {
    if (apiMessage?.trim()) {
      corrDebug('fallback api message (params)', code)
      return apiMessage.trim()
    }
  }

  const strokeN = params?.strokeN ?? 0
  const catalogs = locale === 'ja' ? jaCatalog : enCatalog
  const noStroke = locale === 'ja' ? jaCatalogNoStroke : enCatalogNoStroke

  let template = catalogs[code]
  if (
    (code === 'start_direction' || code === 'end_direction') &&
    strokeN <= 0 &&
    noStroke[code]
  ) {
    template = noStroke[code]!
  }

  // For JA locale without params on glyph-dependent codes, fall back to API EN when glyph missing.
  if (locale === 'ja' && !params?.glyph && (code === 'stroke_order' || code === 'proportions' || code === 'shape')) {
    if (apiMessage?.trim()) {
      corrDebug('fallback api message (no glyph)', code)
      return apiMessage.trim()
    }
  }

  const text = interpolate(template, {
    glyph: params?.glyph ?? '',
    want: params?.want ?? '',
    got: params?.got ?? '',
    strokeN: strokeN > 0 ? strokeN : '',
  })

  const cleaned = text.replace(/\s+/g, ' ').trim()
  corrDebug('map hit', code, locale)
  return cleaned || apiMessage?.trim() || fallbackUnknown[locale]
}

/** English catalog strings for parity tests against Go catalogMessage. */
export function englishCatalogMessage(
  code: CorrectionCode,
  glyph: string,
  want: number,
  got: number,
  strokeN: number,
): string {
  return formatCorrectionDisplay(
    code,
    '',
    { glyph, want, got, strokeN },
    'en',
  )
}
