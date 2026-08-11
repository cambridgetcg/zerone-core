import { knowledgeMainnet } from "./_knowledge";

export const onRequest = (context: Parameters<typeof knowledgeMainnet>[0]) =>
  knowledgeMainnet(context);
