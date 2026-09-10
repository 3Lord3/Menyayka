import { useState } from 'react';

import { App as AntApp, Form } from 'antd';

import { getErrorMessage } from '@shared/lib/errorMessage';
import { EMAIL_PATTERN, PASSWORD_MIN_LENGTH } from '@shared/lib/validation';

import { loginRequest } from '../api/authApi';

import { useApplySession } from './useApplySession';

export interface LoginValues {
  email: string;
  password: string;
}

export function useLogin() {
  const { message } = AntApp.useApp();
  const applySession = useApplySession();
  const [form] = Form.useForm<LoginValues>();
  const email = Form.useWatch('email', form);
  const password = Form.useWatch('password', form);
  const [submitting, setSubmitting] = useState(false);

  // the button gate repeats the form-field rule: a short password doesn't go to a silent 401
  const canSubmit =
    EMAIL_PATTERN.test(email ?? '') && (password?.length ?? 0) >= PASSWORD_MIN_LENGTH;

  async function handleSubmit(values: LoginValues) {
    setSubmitting(true);
    try {
      const session = await loginRequest(values.email.trim(), values.password);
      applySession(session);
    } catch (error) {
      // the backend deliberately doesn't distinguish "no such email" and "wrong password" —
      // always 401
      message.error(getErrorMessage(error, { 401: 'Неверный email или пароль' }));
    } finally {
      setSubmitting(false);
    }
  }

  return { form, submitting, canSubmit, handleSubmit };
}
