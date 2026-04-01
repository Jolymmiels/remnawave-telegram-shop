import adapter from "@sveltejs/adapter-node"

const trustedOrigins = [
  process.env.ORIGIN,
  process.env.FRONTEND_ORIGIN,
  "http://localhost:4321",
  "http://127.0.0.1:4321",
  "http://localhost:8090",
  "http://127.0.0.1:8090"
].filter((value, index, array) => value && array.indexOf(value) === index)

/** @type {import('@sveltejs/kit').Config} */
const config = {
  kit: {
    adapter: adapter(),
    csrf: {
      trustedOrigins
    },
    alias: {
      "@/*": "./src/lib/*"
    }
  }
}

export default config
