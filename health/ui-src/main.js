import { loadTheme } from "./lib/theme.js";
import { setRender } from "./lib/api.js";
import { render, bootIfAuthed } from "./lib/render.js";

loadTheme();
setRender(render);
bootIfAuthed();
