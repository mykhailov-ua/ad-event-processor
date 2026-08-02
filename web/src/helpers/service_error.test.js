import { describe, expect, it } from 'vitest';
import { ApiError, NetworkError } from './api_client.js';
import { mapServiceError } from './service_error.js';

describe('mapServiceError', () => {
  it('maps FORBIDDEN to page block', () => {
    const v = mapServiceError(new ApiError(403, 'FORBIDDEN', 'forbidden'));
    expect(v.kind).toBe('page');
    expect(v.title).toBe('Access denied');
  });

  it('maps CONFLICT to conflict', () => {
    const v = mapServiceError(new ApiError(409, 'CONFLICT', 'already voided'));
    expect(v.kind).toBe('conflict');
  });

  it('maps LEDGER_DRIFT to conflict', () => {
    const v = mapServiceError(new ApiError(409, 'LEDGER_DRIFT', 'drift'));
    expect(v.kind).toBe('conflict');
    expect(v.code).toBe('LEDGER_DRIFT');
  });

  it('maps BAD_REQUEST to inline with server message', () => {
    const v = mapServiceError(new ApiError(400, 'BAD_REQUEST', 'customer_id is required'));
    expect(v.kind).toBe('inline');
    expect(v.message).toBe('customer_id is required');
  });

  it('maps NOT_FOUND to empty', () => {
    const v = mapServiceError(new ApiError(404, 'NOT_FOUND', 'resource not found'));
    expect(v.kind).toBe('empty');
  });

  it('maps RATE_LIMITED to retry with Retry-After', () => {
    const headers = new Headers({ 'Retry-After': '30' });
    const v = mapServiceError(new ApiError(429, 'RATE_LIMITED', 'slow down', false, null, headers));
    expect(v.kind).toBe('retry');
    expect(v.retryAfterSec).toBe(30);
  });

  it('maps NetworkError to toast', () => {
    const v = mapServiceError(new NetworkError('offline'));
    expect(v.kind).toBe('toast');
    expect(v.code).toBe('NETWORK_ERROR');
  });

  it('maps Failed to fetch network errors', () => {
    const v = mapServiceError(new NetworkError('Failed to fetch'));
    expect(v.kind).toBe('toast');
    expect(v.message).toContain('Failed to fetch');
  });

  it('maps 503 payload errors to unavailable message', () => {
    const v = mapServiceError(new ApiError(503, 'UNAVAILABLE', 'down', false, { errors: ['redis', 'ch'] }));
    expect(v.kind).toBe('unavailable');
    expect(v.message).toBe('redis; ch');
  });
});
