const { bytesToHex } = require("viem");
const {
  english,
  generateMnemonic,
  mnemonicToAccount,
} = require("viem/accounts");
let mnemonic,
  account,
  attempt = 0;
do {
  attempt++;
  mnemonic = generateMnemonic(english);
  account = mnemonicToAccount(mnemonic);
  if (account.address.startsWith("0x0")) console.log(attempt, account.address);
} while (!account.address.startsWith("0x0"));
console.log("mnemonic:", mnemonic);
console.log("private:", bytesToHex(account.getHdKey().privKeyBytes));
console.log("address:", account.address);
