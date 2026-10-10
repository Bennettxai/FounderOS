import { app } from "electron";
import path from "path";

// Dev instances must not share userData with the installed app: the packaged
// app and `electron .` both resolve the productName profile dir
// (~/.config/BusinessOS), and the shared singleton lock + SQLite files make
// whichever launches second fail to load (app:// ERR_FAILED). Runs as a
// side-effect import BEFORE any module touches app.getPath("userData").
if (!app.isPackaged) {
  app.setPath("userData", path.join(app.getPath("appData"), "BusinessOS-dev"));
}
