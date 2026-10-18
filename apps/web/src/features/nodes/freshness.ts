// SPDX-License-Identifier: Apache-2.0
type Clock = { wall: number; monotonic: number }
export function observationAge(start: Clock, now: Clock): number {
  return Math.max(0, now.wall - start.wall, now.monotonic - start.monotonic)
}
export function reportIsFresh(node: { freshness: string; fresh_for_millis: number }, elapsed: number, historical: boolean): boolean {
  return !historical && node.freshness === 'fresh' && elapsed < node.fresh_for_millis
}
