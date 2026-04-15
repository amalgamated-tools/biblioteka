import { describe, it, expect } from "vitest";
import { prepareCreationOptions, prepareRequestOptions } from "./passkeys";

describe("prepareCreationOptions", () => {
  it("converts base64url challenge to Uint8Array", () => {
    const options = {
      publicKey: {
        challenge: "dGVzdA", // base64url for "test"
        rp: { name: "Test" },
        user: { id: "dXNlcg", name: "user", displayName: "User" },
        pubKeyCredParams: [],
      },
    };

    const result = prepareCreationOptions(options);

    expect(result.challenge).toBeInstanceOf(Uint8Array);
    expect(Array.from(result.challenge as Uint8Array)).toEqual([
      116, 101, 115, 116,
    ]);
  });

  it("converts base64url user.id to Uint8Array", () => {
    const options = {
      publicKey: {
        challenge: "dGVzdA",
        rp: { name: "Test" },
        user: { id: "dXNlcg", name: "user", displayName: "User" },
        pubKeyCredParams: [],
      },
    };

    const result = prepareCreationOptions(options);
    const user = result.user as PublicKeyCredentialUserEntity;

    expect(user.id).toBeInstanceOf(Uint8Array);
    expect(Array.from(user.id as Uint8Array)).toEqual([117, 115, 101, 114]); // "user"
  });

  it("converts excludeCredentials[].id to Uint8Array", () => {
    const options = {
      publicKey: {
        challenge: "dGVzdA",
        rp: { name: "Test" },
        user: { id: "dXNlcg", name: "user", displayName: "User" },
        pubKeyCredParams: [],
        excludeCredentials: [
          { id: "Y3JlZA", type: "public-key" },
          { id: "Y3JlZDI", type: "public-key" },
        ],
      },
    };

    const result = prepareCreationOptions(options);
    const creds = result.excludeCredentials as PublicKeyCredentialDescriptor[];

    expect(creds).toHaveLength(2);
    expect(creds[0].id).toBeInstanceOf(Uint8Array);
    expect(Array.from(creds[0].id as Uint8Array)).toEqual([99, 114, 101, 100]); // "cred"
  });

  it("handles options without excludeCredentials", () => {
    const options = {
      publicKey: {
        challenge: "dGVzdA",
        rp: { name: "Test" },
        user: { id: "dXNlcg", name: "user", displayName: "User" },
        pubKeyCredParams: [],
      },
    };

    // Should not throw
    const result = prepareCreationOptions(options);
    expect(result.excludeCredentials).toBeUndefined();
  });
});

describe("prepareRequestOptions", () => {
  it("converts base64url challenge to Uint8Array", () => {
    const options = {
      publicKey: {
        challenge: "dGVzdA",
      },
    };

    const result = prepareRequestOptions(options);

    expect(result.challenge).toBeInstanceOf(Uint8Array);
    expect(Array.from(result.challenge as Uint8Array)).toEqual([
      116, 101, 115, 116,
    ]);
  });

  it("converts allowCredentials[].id to Uint8Array", () => {
    const options = {
      publicKey: {
        challenge: "dGVzdA",
        allowCredentials: [{ id: "Y3JlZA", type: "public-key" }],
      },
    };

    const result = prepareRequestOptions(options);
    const creds = result.allowCredentials as PublicKeyCredentialDescriptor[];

    expect(creds).toHaveLength(1);
    expect(creds[0].id).toBeInstanceOf(Uint8Array);
    expect(Array.from(creds[0].id as Uint8Array)).toEqual([99, 114, 101, 100]);
  });

  it("handles options without allowCredentials", () => {
    const options = {
      publicKey: {
        challenge: "dGVzdA",
      },
    };

    const result = prepareRequestOptions(options);
    expect(result.allowCredentials).toBeUndefined();
  });

  it("handles base64url with padding-requiring lengths", () => {
    // "a" = base64url "YQ" (needs 2 pad chars)
    const options = {
      publicKey: {
        challenge: "YQ",
      },
    };

    const result = prepareRequestOptions(options);
    expect(Array.from(result.challenge as Uint8Array)).toEqual([97]); // "a"
  });

  it("handles base64url with special characters", () => {
    // base64url uses - instead of + and _ instead of /
    const options = {
      publicKey: {
        challenge: "ab-_",
      },
    };

    const result = prepareRequestOptions(options);
    expect(result.challenge).toBeInstanceOf(Uint8Array);
  });
});
