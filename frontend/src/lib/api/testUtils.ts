import { vi } from "vitest";
import type { Mock } from "vitest";

export function mockFetchResponse(
  fetchMock: Mock,
  body: unknown,
  status = 200,
) {
  const response = {
    ok: status >= 200 && status < 300,
    status,
    statusText: status === 200 ? "OK" : "Error",
    headers: new Headers({ "content-type": "application/json" }),
    json: vi.fn().mockResolvedValue(body),
    text: vi.fn().mockResolvedValue(JSON.stringify(body)),
  } as unknown as Response;
  fetchMock.mockResolvedValue(response);
}

export function mockNoContentResponse(fetchMock: Mock) {
  fetchMock.mockResolvedValue({
    ok: true,
    status: 204,
    statusText: "No Content",
    headers: new Headers(),
    json: vi.fn(),
    text: vi.fn(),
  } as unknown as Response);
}
