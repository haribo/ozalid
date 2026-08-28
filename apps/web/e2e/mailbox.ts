/**
 * The mailbox the e2e instance sends to.
 *
 * Read over mailpit's API rather than out of the database, so the sending is
 * watched too — it is the half most likely to be misconfigured on a real
 * instance, and the half a database shortcut would never exercise.
 */
import { expect } from '@playwright/test'

const MAILPIT = process.env.OZALID_E2E_MAILPIT ?? 'http://localhost:8045'

/** The link in the newest message sent to an address. */
export async function linkSentTo(address: string): Promise<string> {
  const listed = await fetch(`${MAILPIT}/api/v1/search?query=${encodeURIComponent('to:' + address)}`)
  const { messages } = (await listed.json()) as { messages: { ID: string }[] }
  expect(messages.length, `no message reached ${address}`).toBeGreaterThan(0)

  const body = await fetch(`${MAILPIT}/api/v1/message/${messages[0].ID}`)
  const { Text } = (await body.json()) as { Text: string }
  const found = Text.match(/\/sign-in\/([A-Za-z0-9_-]+)/)
  expect(found, `no sign-in link in the message:\n${Text}`).not.toBeNull()
  return found![1]
}
