import { describe, expect, it } from 'vitest'
import { applyIncomingMessage, type Stroke } from '../../src/services/strokeSync'

function pendingStroke(overrides: Partial<Stroke> = {}): Stroke {
  return {
    points: [
      { x: 0, y: 0 },
      { x: 10, y: 10 },
    ],
    color: '#1d4ed8',
    width: 4,
    clientId: 'local-client',
    startedAtUnixMs: 1_700_000_000_000,
    ...overrides,
  }
}

describe('applyIncomingMessage personal stroke isolation', () => {
  it('merges server id onto a matching pending local stroke', () => {
    const local = pendingStroke()
    const result = applyIncomingMessage([local], {
      type: 'stroke',
      stroke: { ...local, id: 42 },
    })

    expect(result.action, 'expected echo to merge id onto pending local stroke').toBe('merged-id')
    expect(result.strokes).toHaveLength(1)
    expect(result.strokes[0].id).toBe(42)
  })

  it('ignores an incoming stroke that does not match local pending state', () => {
    const local = pendingStroke({ id: 1 })
    const foreign = pendingStroke({
      clientId: 'other-client',
      startedAtUnixMs: 99,
      id: 7,
    })
    const result = applyIncomingMessage([local], { type: 'stroke', stroke: foreign })

    expect(
      result.action,
      'expected foreign live stroke to be ignored (no shared multi-user canvas)',
    ).toBe('ignored-foreign-stroke')
    expect(result.strokes, 'canvas must not grow from unmatched inbound stroke').toEqual([local])
  })

  it('ignores delete for an unknown id (no-op)', () => {
    const local = pendingStroke({ id: 5 })
    const result = applyIncomingMessage([local], { type: 'delete', delete: 999 })

    expect(
      result.action,
      'expected unknown delete id to be a no-op under personal isolation',
    ).toBe('ignored-unknown-delete')
    expect(result.strokes).toEqual([local])
  })

  it('removes a stroke when delete id is known locally', () => {
    const keep = pendingStroke({ id: 1, clientId: 'a' })
    const remove = pendingStroke({ id: 2, clientId: 'b', startedAtUnixMs: 2 })
    const result = applyIncomingMessage([keep, remove], { type: 'delete', delete: 2 })

    expect(result.action).toBe('deleted')
    expect(result.strokes).toEqual([keep])
  })
})
