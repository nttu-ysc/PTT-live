import './style.css';
import './app.css';

import './commponents/login.js';
import './commponents/board.js';
import './commponents/post.js';
import './commponents/post-detail.js';

import { CheckUpdate, OpenURL, GetVersion, PerformUpdate, Quit, Restart } from '../wailsjs/go/main/App.js';
import { EventsOn } from '../wailsjs/runtime/runtime.js';

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

    // ── About modal ────────────────────────────────────────────────────────────
    const aboutModal = document.getElementById('about-modal');
    const aboutOverlay = document.getElementById('about-overlay');
    const aboutCloseBtn = document.getElementById('about-close-btn');
    const aboutVersionEl = document.getElementById('about-version');

    function showAbout() {
        GetVersion().then(v => {
            if (aboutVersionEl) aboutVersionEl.textContent = `v${v}`;
        }).catch(() => {});
        aboutModal.classList.add('visible');
        aboutOverlay.classList.add('visible');
    }

    function hideAbout() {
        aboutModal.classList.remove('visible');
        aboutOverlay.classList.remove('visible');
    }

    aboutCloseBtn.addEventListener('click', hideAbout);
    aboutOverlay.addEventListener('click', hideAbout);

    document.getElementById('about-github-link').addEventListener('click', (e) => {
        e.preventDefault();
        OpenURL('https://github.com/nttu-ysc/PTT-live');
    });

    document.getElementById('about-releases-link').addEventListener('click', (e) => {
        e.preventDefault();
        OpenURL('https://github.com/nttu-ysc/PTT-live/releases');
    });

    // Listen for menu events from Go backend
    EventsOn('show-about', showAbout);
    EventsOn('check-update-menu', () => { performUpdateCheck(true); });

    // ── Update toast ───────────────────────────────────────────────────────────
    const updateToast = document.getElementById('update-toast');
    const updateToastMsg = document.getElementById('update-toast-msg');
    const updateToastClose = document.getElementById('update-toast-close');
    const updateToastAction = document.getElementById('update-toast-action');

    function performUpdateCheck(fromMenu = false) {
        CheckUpdate().then(info => {
            if (info.hasUpdate === 'true') {
                updateToastMsg.textContent = `🎉 有新版本 ${info.latestVersion} 可供更新！`;
                updateToastAction.textContent = '立即更新';
                updateToastAction.style.display = 'block';
                updateToastAction.disabled = false;
                
                updateToastAction.onclick = () => {
                    updateToastAction.textContent = '正在更新中...';
                    updateToastAction.disabled = true;
                    updateToastClose.style.display = 'none'; // hide close button during update
                    
                    PerformUpdate().then(res => {
                        if (res === 'ok') {
                            updateToastMsg.textContent = '✅ 更新成功！即將重新啟動應用程式...';
                            updateToastAction.style.display = 'none';
                            setTimeout(() => {
                                Restart();
                            }, 2000);
                        } else {
                            updateToastMsg.textContent = `❌ ${res}`;
                            updateToastAction.textContent = '手動下載';
                            updateToastAction.disabled = false;
                            updateToastClose.style.display = 'block';
                            updateToastAction.onclick = () => {
                                OpenURL(info.url);
                                updateToast.classList.remove('visible');
                            };
                        }
                    }).catch(err => {
                        updateToastMsg.textContent = `❌ 更新失敗: ${err}`;
                        updateToastAction.textContent = '手動下載';
                        updateToastAction.disabled = false;
                        updateToastClose.style.display = 'block';
                        updateToastAction.onclick = () => {
                            OpenURL(info.url);
                            updateToast.classList.remove('visible');
                        };
                    });
                };
                updateToast.classList.add('visible');
            } else if (fromMenu) {
                updateToastMsg.textContent = '✅ 您目前已是最新版本！';
                updateToastAction.style.display = 'none';
                updateToast.classList.add('visible');
                setTimeout(() => updateToast.classList.remove('visible'), 3000);
            }
        }).catch(err => {
            console.error('CheckUpdate failed:', err);
            if (fromMenu) {
                updateToastMsg.textContent = '⚠️ 無法檢查更新，請確認網路連線。';
                updateToastAction.style.display = 'none';
                updateToast.classList.add('visible');
                setTimeout(() => updateToast.classList.remove('visible'), 3000);
            }
        });
    }

    updateToastClose.addEventListener('click', () => {
        updateToast.classList.remove('visible');
    });

    // Check for updates on startup (after a short delay)
    setTimeout(() => performUpdateCheck(false), 2000);
});
