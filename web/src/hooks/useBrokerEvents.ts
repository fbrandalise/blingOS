import { useEffect } from "react";
import { useQueryClient } from "@tanstack/react-query";

import { subscribeBrokerEvents } from "../api/brokerEvents";
import { useAppStore } from "../stores/app";

export function useBrokerEvents(enabled: boolean) {
  const queryClient = useQueryClient();
  const setBrokerConnected = useAppStore((s) => s.setBrokerConnected);

  useEffect(() => {
    if (!enabled) return;

    const unsub = subscribeBrokerEvents({
      ready: () => setBrokerConnected(true),
      message: () => {
        void queryClient.invalidateQueries({ queryKey: ["messages"] });
        void queryClient.invalidateQueries({ queryKey: ["thread-messages"] });
        void queryClient.invalidateQueries({ queryKey: ["office-members"] });
        void queryClient.invalidateQueries({ queryKey: ["channel-members"] });
      },
      activity: () => {
        void queryClient.invalidateQueries({ queryKey: ["office-members"] });
        void queryClient.invalidateQueries({ queryKey: ["channel-members"] });
      },
      office_changed: () => {
        void queryClient.invalidateQueries({ queryKey: ["channels"] });
        void queryClient.invalidateQueries({ queryKey: ["office-members"] });
        void queryClient.invalidateQueries({ queryKey: ["channel-members"] });
      },
      action: () => {
        void queryClient.invalidateQueries({ queryKey: ["actions"] });
        void queryClient.invalidateQueries({ queryKey: ["office-tasks"] });
      },
    });

    return unsub;
  }, [enabled, queryClient, setBrokerConnected]);
}
