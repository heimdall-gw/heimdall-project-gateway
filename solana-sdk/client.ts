import { 
  Connection, PublicKey, Transaction 
} from "@solana/web3.js";
import type { Commitment } from "@solana/web3.js";
import { LRUCache } from "lru-cache";

type LRUType = LRUCache<any, any>;

type ClientOptions = {
  rpcUrl: string;
  commitment?: Commitment;
  cacheTTLms?: number;
  retryAttempts?: number;
};

export class SolanaClient {
    connection: Connection;
    rpcUrl: string;
    cache: LRUType;
    retryAttempts: number;

  constructor(opts: ClientOptions) {
    this.rpcUrl = opts.rpcUrl;
    this.connection = new Connection(opts.rpcUrl, opts.commitment ?? "confirmed");
    this.cache = new LRUCache({ max: 5000, ttl: opts.cacheTTLms ?? 30_000 }) as LRUType;
    this.retryAttempts = opts.retryAttempts ?? 4;
  }

  private keyForCache(prefix: string, payload: string) {
    return `${prefix}:${payload}`;
  }

  private async withRetry<T>(fn: () => Promise<T>, attempts = this.retryAttempts): Promise<T> {
    let i = 0;
    while (true) {
      try {
        return await fn();
      } catch (err) {
        i++;
        if (i >= attempts) throw err;
        const delay = 200 * i; 
        await new Promise((r) => setTimeout(r, delay));
      }
    }
  }

    async getAccountInfo(address: string) {
    const key = this.keyForCache("accountInfo", address);
    const cached = this.cache.get(key);
    if (cached) return cached;
    const pk = new PublicKey(address);
    const info = await this.withRetry(() => this.connection.getAccountInfo(pk));
    this.cache.set(key, info);
    return info;
  }

  async getBalance(address: string) {
    const key = this.keyForCache("balance", address);
    const cached = this.cache.get(key);
    if (cached) return cached;
    const pk = new PublicKey(address);
    const balance = await this.withRetry(() => this.connection.getBalance(pk));
    this.cache.set(key, balance);
    return balance;
  }

  async getBlock(slot: number, opts = { maxSupportedTransactionVersion: 0 }) {
    const key = this.keyForCache("block", String(slot));
    const cached = this.cache.get(key);
    if (cached) return cached;
    const block = await this.withRetry(() => this.connection.getBlock(slot, opts));
    this.cache.set(key, block);
    return block;
  }

  async getBlocks(fromSlot: number, toSlot: number, opts = { maxSupportedTransactionVersion: 0 }) {
    if (toSlot < fromSlot) throw new Error("toSlot deve ser >= fromSlot");
    const blocks: Array<any> = [];
    const MAX_BATCH = 500;
    if (toSlot - fromSlot + 1 > MAX_BATCH) {
      throw new Error(`Range muito grande. Máx ${MAX_BATCH} slots por chamada.`);
    }
    for (let s = fromSlot; s <= toSlot; s++) {
      const blk = await this.getBlock(s, opts);
      blocks.push(blk ?? null);
    }
    return blocks;
  }

  async getHealth(): Promise<string | number> {
    const key = this.keyForCache("health", "v1");
    const cached = this.cache.get(key);
    if (cached) return cached;
    const health = await this.withRetry(async () => {
      const anyConn: any = this.connection as any;
      if (typeof anyConn.getHealth === "function") {
        return await anyConn.getHealth();
      } else {
        const s = await this.connection.getSlot();
        return s ? "ok" : "unknown";
      }
    });
    this.cache.set(key, health);
    return health;
  }

  subscribeLogs(programId: string, callback: (logInfo: any) => void): () => Promise<void> {
    const pid = new PublicKey(programId);
    const subId = this.connection.onLogs(
      pid,
      (logInfo: any) => {
        try {
          callback(logInfo);
        } catch (e) {
          console.error("subscribe callback error:", e);
        }
      },
      "confirmed"
    );

    return async () => {
      try {
        this.connection.removeOnLogsListener(subId);
      } catch (e) {
      }
    };
  }

    async getTransaction(signature: string): Promise<any> {
    const key = this.keyForCache("tx", signature);
    const cached = this.cache.get(key);
    if (cached) return cached;

    const tx = await this.withRetry(() => this.connection.getTransaction(signature, { maxSupportedTransactionVersion: 0 }));
    this.cache.set(key, tx);
    return tx;
    }
}
