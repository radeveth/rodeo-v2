import fs from "fs";
import { keccak256, solidityPackedKeccak256 } from "ethers";
import { MerkleTree } from "merkletreejs";

const ONE = 1_000_000_000_000_000_000n;

(async () => {
  const distributions = fs
    .readFileSync("sss.csv", { encoding: "utf8" })
    .split("\n")
    .filter((n) => n)
    .map((l) => l.split(","))
    .map(([user, amount]) => ({
      user,
      amount: (BigInt(amount.split(".")[0]) * ONE).toString(),
    }));
  let total = 0n;
  distributions.forEach((d) => {
    total += BigInt(d.amount);
  });
  /*
  distributions.forEach((d) => {
    d.amount = ((BigInt(d.amount) * 240000n * ONE) / total).toString();
  });
  */
  console.log(total);
  console.log(distributions.reduce((t, d) => t + BigInt(d.amount), 0n));
  const leafs = [];
  let users = [];
  for (let u of distributions) {
    leafs.push(
      solidityPackedKeccak256(["address", "uint256"], [u.user, u.amount])
    );
    users.push(u);
  }
  const merkleTree = new MerkleTree(leafs, keccak256, {
    sort: true,
  });
  const root = merkleTree.getHexRoot();
  users = users.map((u, i) => ({
    ...u,
    proof: merkleTree.getHexProof(leafs[i]),
  }));
  const week = "8";
  fs.writeFileSync(
    `assets/stip/${week}.json`,
    JSON.stringify({ week, root, users }, null, 2)
  );
  console.log({ root, week, count: users.length });
})();
