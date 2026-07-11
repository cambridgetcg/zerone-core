import { proxyMainnet } from "../_proxy";

export const onRequest = (context: Parameters<typeof proxyMainnet>[0]) =>
  proxyMainnet(context, "rpc");
