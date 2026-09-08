export type RecognizeAcceptInput = {
  attempt: number
  currentAttempt: number
  responseBoardRev: number
  requestedBoardRev: number
  localBoardRev: number
}

/**
 * Drop late/stale recognize HTTP responses when the board moved or a newer
 * request superseded this attempt.
 */
export function shouldAcceptRecognizeResponse(input: RecognizeAcceptInput): boolean {
  return (
    input.attempt === input.currentAttempt &&
    input.responseBoardRev === input.requestedBoardRev &&
    input.localBoardRev === input.requestedBoardRev
  )
}
