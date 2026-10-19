// SPDX-License-Identifier: Apache-2.0
// Initial extraction of the old async predicate's single-read behavior; replace
// only after observing tests-first RED against real HTTP and native browser.
export async function waitForOperation(readIdentity, operation) {
 return (await readIdentity()).operations.includes(operation)
}
