// solana-sdk/test.ts
import * as web3 from "@solana/web3.js";

(async () => {
  try {
    const RPC = "https://api.mainnet-beta.solana.com"; 
    const conn = new web3.Connection(RPC, "confirmed");

    const latest = await conn.getLatestBlockhash();
    console.log("Último blockhash:", latest.blockhash);
    const slot = await conn.getSlot();
    console.log("Slot atual:", slot);

    const block = await conn.getBlock(slot, { maxSupportedTransactionVersion: 0 });
    if (!block) {
      console.log("Bloco retornou null (pode ser que o provider não tenha histórico desse slot).");
      return;
    }
    console.log("Transações no bloco:", block.transactions.length);

    let candidatePubkey: string | null = null;

    for (const tx of block.transactions) {
      try {
        const message: any = tx.transaction?.message;
        if (!message) continue;

        if (message.accountKeys && message.accountKeys.length > 0) {
          const first = message.accountKeys[0];
          if (typeof first === "string") {
            candidatePubkey = first;
          } else if (first?.pubkey) {
            candidatePubkey = first.pubkey.toString ? first.pubkey.toString() : String(first.pubkey);
          } else if (first?.toBase58) {
            candidatePubkey = first.toBase58();
          } else if (first instanceof web3.PublicKey) {
            candidatePubkey = first.toBase58();
          }
        }

        if (!candidatePubkey) {
          const sig = (tx.transaction?.signatures && tx.transaction.signatures[0]) || null;
          if (sig) {
            const detailed = await conn.getTransaction(sig);
            if (detailed && detailed.transaction && detailed.transaction.message && detailed.transaction.message.accountKeys) {
              const fk = detailed.transaction.message.accountKeys[0];
              if (typeof fk === "string") candidatePubkey = fk;
              else if (fk?.toBase58) candidatePubkey = fk.toBase58();
              else if (fk?.pubkey) candidatePubkey = fk.pubkey.toString ? fk.pubkey.toString() : String(fk.pubkey);
            }
          }
        }
      } catch (e) {
      }
      if (candidatePubkey) break;
    }

    if (!candidatePubkey) {
      console.log("Não foi possível extrair nenhuma PublicKey do bloco. Tenta com outro bloco ou outro RPC.");
      return;
    }
    console.log("PublicKey encontrada (candidate):", candidatePubkey);

    // 4) valida e usa a PublicKey: getAccountInfo + getBalance
    let pk: web3.PublicKey;
    try {
      pk = new web3.PublicKey(candidatePubkey);
    } catch (e) {
      console.log("PublicKey inválida extraída:", candidatePubkey);
      return;
    }

    const info = await conn.getAccountInfo(pk);
    const balance = await conn.getBalance(pk);

    console.log("=== Resultado para a PublicKey extraída ===");
    console.log("Address:", pk.toBase58());
    console.log("Balance (lamports):", balance);
    console.log("Balance (SOL):", balance / 1e9);
    console.log("AccountInfo:", info ? {
      lamports: info.lamports,
      owner: info.owner?.toString(),
      executable: info.executable,
      dataLen: info.data?.length ?? 0
    } : null);

  } catch (err: any) {
    console.error("Erro no teste:", err?.message ?? err);
  }
})();
