import { knowledgeMainnet } from "./_knowledge";

type ApiMiddlewareContext = Parameters<typeof knowledgeMainnet>[0] & {
  next(): Promise<Response>;
};

const KNOWLEDGE_PATH = "/api/knowledge";
const MAX_PATH_DECODES = 4;

function isKnowledgeNamespace(pathname: string): boolean {
  let candidate = pathname;

  for (let pass = 0; pass <= MAX_PATH_DECODES; pass += 1) {
    if (
      candidate === KNOWLEDGE_PATH ||
      candidate.startsWith(`${KNOWLEDGE_PATH}/`) ||
      candidate.startsWith(`${KNOWLEDGE_PATH}%`)
    ) {
      return true;
    }
    if (pass === MAX_PATH_DECODES) {
      break;
    }

    let decoded: string;
    try {
      decoded = decodeURIComponent(candidate);
    } catch {
      return false;
    }
    if (decoded === candidate) {
      break;
    }
    candidate = decoded;
  }

  return false;
}

// Pages route matching keeps encoded slashes inside a segment. Intercept the
// knowledge namespace before static SPA fallback, including bounded nested
// encodings, while leaving every unrelated API route unchanged.
export const onRequest = (context: ApiMiddlewareContext) => {
  const pathname = new URL(context.request.url).pathname;
  return isKnowledgeNamespace(pathname)
    ? knowledgeMainnet(context)
    : context.next();
};
