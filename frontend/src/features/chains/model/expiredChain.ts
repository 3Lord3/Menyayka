import { isAxiosError } from 'axios';

export const EXPIRED_CHAIN_MESSAGE = 'Время истекло, цепочка распалась';

// the (answer or quick-replacement) deadline has passed and the server already rolled the
// chain back — nothing to confirm/select
export function isChainExpired(error: unknown): boolean {
  return isAxiosError(error) && error.response?.status === 410;
}
