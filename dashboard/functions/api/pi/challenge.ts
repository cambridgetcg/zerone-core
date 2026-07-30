import { runPiEndpoint } from "./_service";

export const onRequest = (
  context: Parameters<typeof runPiEndpoint>[1],
): Promise<Response> => runPiEndpoint("challenge", context);
