import './style.css';
import './app.css';

import './commponents/login.js';
import './commponents/board.js';
import './commponents/post.js';
import './commponents/post-detail.js';
import {WindowFullscreen} from "../wailsjs/runtime";

// document.oncontextmenu = new Function('event.returnValue=false');
document.addEventListener('DOMContentLoaded', () => {
    console.log('Wails ready!');
});

