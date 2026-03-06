import './style.css';
import './app.css';

import './commponents/login.js';
import './commponents/board.js';
import './commponents/post.js';
import './commponents/post-detail.js';

document.addEventListener('DOMContentLoaded', () => {
    const slider = document.getElementById('opacity-slider');

    // Directly control the <html> element's background-color alpha.
    // html has a default background-color: rgba(0,0,0,1) in style.css.
    // By changing the alpha via inline style, we override the CSS rule
    // regardless of hover state, making the effect immediate and persistent.
    slider.addEventListener('input', () => {
        const alpha = parseInt(slider.value, 10) / 100;
        document.documentElement.style.backgroundColor = `rgba(0, 0, 0, ${alpha})`;
    });
});
