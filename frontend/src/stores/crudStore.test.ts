import { describe, it, expect, beforeEach, vi } from "vitest";
import { CrudStore } from "./crudStore.svelte";

interface TestEntity {
  id: string;
  name: string;
}

interface TestInput {
  name: string;
}

const fakeEntity: TestEntity = { id: "e1", name: "Test" };
const fakeEntity2: TestEntity = { id: "e2", name: "Other" };

const listFn = vi.fn<() => Promise<TestEntity[]>>();
const createFn = vi.fn<(input: TestInput) => Promise<TestEntity>>();
const updateFn = vi.fn<(id: string, input: TestInput) => Promise<TestEntity>>();
const deleteFn = vi.fn<(id: string) => Promise<void>>();

function makeStore() {
  return new CrudStore<TestEntity, TestInput>({
    list: listFn,
    create: createFn,
    update: updateFn,
    delete: deleteFn,
  });
}

describe("CrudStore", () => {
  let store: CrudStore<TestEntity, TestInput>;

  beforeEach(() => {
    vi.clearAllMocks();
    store = makeStore();
  });

  describe("load", () => {
    it("fetches items and sets loaded", async () => {
      listFn.mockResolvedValue([fakeEntity]);

      await store.load();

      expect(listFn).toHaveBeenCalledTimes(1);
      expect(store.items).toEqual([fakeEntity]);
      expect(store.loaded).toBe(true);
      expect(store.loading).toBe(false);
    });

    it("does not call API again after already loaded", async () => {
      listFn.mockResolvedValue([fakeEntity]);

      await store.load();
      await store.load();

      expect(listFn).toHaveBeenCalledTimes(1);
    });

    it("calls API only once when invoked concurrently", async () => {
      listFn.mockResolvedValue([fakeEntity]);

      await Promise.all([store.load(), store.load(), store.load()]);

      expect(listFn).toHaveBeenCalledTimes(1);
      expect(store.items).toEqual([fakeEntity]);
      expect(store.loaded).toBe(true);
    });

    it("resets loading on API error", async () => {
      listFn.mockRejectedValue(new Error("fail"));

      await store.load();

      expect(store.loading).toBe(false);
      expect(store.loaded).toBe(false);
      expect(store.items).toEqual([]);
    });
  });

  describe("add", () => {
    it("appends the created entity to items", async () => {
      createFn.mockResolvedValue(fakeEntity);

      const result = await store.add({ name: "Test" });

      expect(result).toEqual(fakeEntity);
      expect(store.items).toEqual([fakeEntity]);
    });

    it("appends to existing items", async () => {
      store.items = [fakeEntity];
      createFn.mockResolvedValue(fakeEntity2);

      await store.add({ name: "Other" });

      expect(store.items).toEqual([fakeEntity, fakeEntity2]);
    });
  });

  describe("edit", () => {
    it("replaces the updated entity in items", async () => {
      const updated: TestEntity = { id: "e1", name: "Updated" };
      store.items = [fakeEntity, fakeEntity2];
      updateFn.mockResolvedValue(updated);

      const result = await store.edit("e1", { name: "Updated" });

      expect(result).toEqual(updated);
      expect(store.items).toEqual([updated, fakeEntity2]);
    });
  });

  describe("remove", () => {
    it("removes the entity from items", async () => {
      store.items = [fakeEntity, fakeEntity2];
      deleteFn.mockResolvedValue(undefined);

      await store.remove("e1");

      expect(deleteFn).toHaveBeenCalledWith("e1");
      expect(store.items).toEqual([fakeEntity2]);
    });

    it("propagates errors and leaves items unchanged", async () => {
      store.items = [fakeEntity, fakeEntity2];
      deleteFn.mockRejectedValue(new Error("fail"));

      await expect(store.remove("e1")).rejects.toThrow("fail");
      expect(store.items).toEqual([fakeEntity, fakeEntity2]);
    });
  });

  describe("add error propagation", () => {
    it("propagates errors and leaves items unchanged", async () => {
      store.items = [fakeEntity];
      createFn.mockRejectedValue(new Error("fail"));

      await expect(store.add({ name: "Bad" })).rejects.toThrow("fail");
      expect(store.items).toEqual([fakeEntity]);
    });
  });

  describe("edit error propagation", () => {
    it("propagates errors and leaves items unchanged", async () => {
      store.items = [fakeEntity, fakeEntity2];
      updateFn.mockRejectedValue(new Error("fail"));

      await expect(store.edit("e1", { name: "Bad" })).rejects.toThrow("fail");
      expect(store.items).toEqual([fakeEntity, fakeEntity2]);
    });
  });
});
