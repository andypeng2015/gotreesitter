// Function declarations with various patterns
function processFile(path, maxSize) {
  const data = require("fs").readFileSync(path);
  if (data.length > maxSize) {
    throw new Error(`File too large: ${data.length} > ${maxSize}`);
  }
  return data;
}

// Arrow functions and destructuring
const parseOptions = ({ language, timeout = 30, ...rest }) => ({
  lang: language,
  maxTime: timeout,
  extra: rest,
});

// Class with inheritance, getters/setters, private fields
class BaseParser {
  #language;
  #cache = new Map();

  constructor(language, options = {}) {
    this.#language = language;
    this.timeout = options.timeout ?? 30;
  }

  get language() {
    return this.#language;
  }

  parse(source) {
    throw new Error("Not implemented");
  }
}

class AdvancedParser extends BaseParser {
  #maxRetries;

  constructor(language, { maxRetries = 3, ...options } = {}) {
    super(language, options);
    this.#maxRetries = maxRetries;
  }

  async parse(source) {
    const cacheKey = Buffer.from(source).toString("base64").slice(0, 32);
    if (this.#cache?.has(cacheKey)) {
      return this.#cache.get(cacheKey);
    }

    for (let attempt = 0; attempt < this.#maxRetries; attempt++) {
      try {
        const tree = await this.#doParse(source);
        this.#cache?.set(cacheKey, tree);
        return tree;
      } catch (err) {
        if (attempt === this.#maxRetries - 1) throw err;
        await new Promise((r) => setTimeout(r, 100 * (attempt + 1)));
      }
    }
  }

  async #doParse(source) {
    return { root: null, source, hasError: false };
  }
}

// Template literals with nesting
function formatNode(node, depth = 0) {
  const indent = " ".repeat(depth * 2);
  const children = node.children
    .map((c) => formatNode(c, depth + 1))
    .join("\n");
  return `${indent}(${node.kind} [${node.startByte}-${node.endByte}]${
    children ? `\n${children}\n${indent}` : ""
  })`;
}

// Destructuring, spread, optional chaining
function analyzeTree(tree) {
  const { root, source, hasError } = tree;
  const nodes = [...walkTree(root)];
  const functions = nodes.filter((n) => n.kind === "function_definition");

  return {
    totalNodes: nodes.length,
    functions: functions.length,
    hasError,
    rootKind: root?.kind ?? "unknown",
    firstChild: root?.children?.[0]?.kind,
  };
}

// Generator function
function* walkTree(node) {
  yield node;
  for (const child of node.children ?? []) {
    yield* walkTree(child);
  }
}

// Promises and async/await
async function parseFiles(paths, parser) {
  const results = await Promise.allSettled(
    paths.map(async (path) => {
      const source = await require("fs").promises.readFile(path);
      return parser.parse(source);
    })
  );

  return results.reduce(
    (acc, result, i) => {
      if (result.status === "fulfilled") {
        acc.success.push({ path: paths[i], tree: result.value });
      } else {
        acc.failed.push({ path: paths[i], error: result.reason.message });
      }
      return acc;
    },
    { success: [], failed: [] }
  );
}

// Dynamic import
async function loadLanguage(name) {
  const mod = await import(`./languages/${name}.js`);
  return mod.default ?? mod;
}

// Proxy and Reflect
function createNodeProxy(node) {
  return new Proxy(node, {
    get(target, prop, receiver) {
      if (prop === "childByFieldName") {
        return (name) =>
          target.children.find((c) => c.kind === name) ?? null;
      }
      return Reflect.get(target, prop, receiver);
    },
  });
}

// Export patterns
export { AdvancedParser, parseFiles };
export default BaseParser;
