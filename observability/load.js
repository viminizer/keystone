// Clients that use keystone like a cache: mostly reads, some writes with a
// time limit, a few deletes, and a few bad commands.
//
//   k6 run observability/load.js

import redis from "k6/experimental/redis";

const client = new redis.Client("redis://localhost:6380");

export const options = {
  stages: [
    { duration: "1m", target: 20 },
    { duration: "3m", target: 20 },
    { duration: "1m", target: 100 },
    { duration: "2m", target: 20 },
    { duration: "30s", target: 0 },
  ],
};

const keyCount = 10000;
const value = "x".repeat(100);

function randomKey() {
  return "user:" + Math.floor(Math.random() * keyCount);
}

export default async function () {
  const roll = Math.random();
  try {
    if (roll < 0.7) {
      await client.get(randomKey());
    } else if (roll < 0.8) {
      await client.set(randomKey(), value, 0);
    } else if (roll < 0.9) {
      const seconds = 1 + Math.floor(Math.random() * 30);
      await client.set(randomKey(), value, seconds);
    } else if (roll < 0.95) {
      await client.del(randomKey());
    } else if (roll < 0.98) {
      await client.ttl(randomKey());
    } else {
      await client.sendCommand("GET");
    }
  } catch (error) {
    // A missing key and a bad command both come back as errors here.
  }
}
