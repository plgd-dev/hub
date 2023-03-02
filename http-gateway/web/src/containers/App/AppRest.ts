import { fetchApi, security } from "@shared-ui/common/services";
import { SecurityConfig } from "@/containers/App/App.types";

export const getAppWellKnownConfiguration = (wellKnowConfigUrl: string) => {
  const { cancelRequestDeadlineTimeout } = security.getGeneralConfig() as SecurityConfig
  return fetchApi(`${wellKnowConfigUrl}/.well-known/configuration`, {
    useToken: false,
    cancelRequestDeadlineTimeout
  });
}
