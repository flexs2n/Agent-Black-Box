import { describe, it, expect } from 'vitest';

describe('web smoke test', () => {
  it('api module exports expected types', async () => {
    const mod = await import('../src/lib/api');
    expect(mod).toBeDefined();
    expect(typeof mod.api.listTraces).toBe('function');
    expect(typeof mod.api.getTrace).toBe('function');
    expect(typeof mod.api.computeDiff).toBe('function');
    expect(typeof mod.api.getDiff).toBe('function');
    expect(typeof mod.api.listProjects).toBe('function');
    expect(typeof mod.api.createProject).toBe('function');
    expect(typeof mod.api.createApiKey).toBe('function');
    expect(typeof mod.api.listApiKeys).toBe('function');
    expect(typeof mod.api.deleteApiKey).toBe('function');
  });

  it('home page component exports', async () => {
    const mod = await import('../src/app/page');
    expect(mod).toBeDefined();
    expect(typeof mod.default).toBe('function');
  });
});