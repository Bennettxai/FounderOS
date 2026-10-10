import type { PageLoad } from "./$types";
import { redirect } from "@sveltejs/kit";

// /communication has no landing view of its own — it is a tabbed hub.
// Send visitors to the Email (unified inbox) tab by default.
export const load: PageLoad = () => {
  throw redirect(307, "/communication/email");
};

export const ssr = false;
