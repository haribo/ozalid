/**
 * The mailbox the e2e instance sends to.
 *
 * Read over mailpit's API rather than out of the database, so the sending is
 * watched too — it is the half most likely to be misconfigured on a real
 * instance, and the half a database shortcut would never exercise.
 */
import { expect } from '@playwright/test'

const MAILPIT = process.env.OZALID_E2E_MAILPIT ?? 'http://localhost:8045'

/**
 * Throw away what an address has already been sent.
 *
 * Called before asking for a link, so that waiting for "a message" is waiting
 * for *this* message. Without it a reader picks up an older one — already spent,
 * since a link works once — and the sign-in fails for a reason that has nothing
 * to do with what the test is about.
 */
export async function emptyMailbox(address: string): Promise<void> {
  const listed = await fetch(
    `${MAILPIT}/api/v1/search?query=${encodeURIComponent('to:' + address)}`,
  )
  const { messages } = (await listed.json()) as { messages: { ID: string }[] }
  if (!messages.length) return
  await fetch(`${MAILPIT}/api/v1/messages`, {
    method: 'DELETE',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify({ IDs: messages.map((m) => m.ID) }),
  })
}

/**
 * The link in the newest message sent to an address.
 *
 * Polled, not sampled: the server answers before the message leaves, on purpose
 * — waiting for the send is what let a stopwatch tell an address with an
 * account from one without (#92). A reader that looks once races the send, and
 * would pass on a fast machine and fail in CI.
 *
 * Pair it with `emptyMailbox` before the ask, or it may hand back a link from
 * an earlier one.
 */
export async function linkSentTo(address: string): Promise<string> {
  let messages: { ID: string }[] = []
  await expect
    .poll(
      async () => {
        const listed = await fetch(
          `${MAILPIT}/api/v1/search?query=${encodeURIComponent('to:' + address)}`,
        )
        messages = ((await listed.json()) as { messages: { ID: string }[] }).messages
        return messages.length
      },
      { message: `no message reached ${address}` },
    )
    .toBeGreaterThan(0)

  const body = await fetch(`${MAILPIT}/api/v1/message/${messages[0].ID}`)
  const { Text } = (await body.json()) as { Text: string }
  const found = Text.match(/\/sign-in\/([A-Za-z0-9_-]+)/)
  expect(found, `no sign-in link in the message:\n${Text}`).not.toBeNull()
  return found![1]
}
