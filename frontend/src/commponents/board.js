import { GotoBoard, GetHotBoards } from '../../wailsjs/go/pttclient/PttClient.js';
import { displayPosts, setCurrentBoard } from './post.js';

class BoardPage extends HTMLElement {
  connectedCallback() {
    this.innerHTML = `
            <div class="page">
                <h2>請輸入看板</h2>
                <form id="board-form">
                    <div id="error-message" style="color: red;"></div>
                    <div class="input-box">
                        <label for="boardName">看板名稱:</label>
                        <input type="text" id="boardName" class="input" autocapitalize="none" autocomplete="off" />
                    </div>
                    <div class="input-box">
                        <button type="submit" class="btn">送出</button>
                    </div>
                </form>
                <div class="hot-boards">
                    <div class="hot-boards-header">
                        <div class="hot-boards-title">🔥 熱門看板</div>
                        <button class="hot-boards-refresh-btn" id="hot-boards-refresh-btn" title="重新整理熱門看板">
                            <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
                                <polyline points="23 4 23 10 17 10"></polyline>
                                <polyline points="1 20 1 14 7 14"></polyline>
                                <path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"></path>
                            </svg>
                        </button>
                    </div>
                    <div class="chips-wrap" id="chips-wrap">
                        <span style="font-size:12px;color:rgba(255,255,255,0.35);">載入中…</span>
                    </div>
                </div>
            </div>
        `;

    this.querySelector('#boardName').focus();

    // ── Hot boards loader ────────────────────────────────────────────────────
    const loadHotBoards = () => {
      const wrap = this.querySelector('#chips-wrap');
      const refreshBtn = this.querySelector('#hot-boards-refresh-btn');
      if (!wrap) return;

      wrap.innerHTML = '<span style="font-size:12px;color:rgba(255,255,255,0.35);">載入中…</span>';
      if (refreshBtn) refreshBtn.classList.add('spinning');

      GetHotBoards().then(boards => {
        if (refreshBtn) refreshBtn.classList.remove('spinning');
        if (!wrap) return;
        if (!boards || boards.length === 0) {
          wrap.innerHTML = '<span style="font-size:12px;color:rgba(255,255,255,0.3);">無法載入</span>';
          return;
        }
        wrap.innerHTML = '';
        boards.forEach(board => {
          const chip = document.createElement('button');
          chip.type = 'button';
          chip.className = 'board-chip';
          chip.innerHTML = `${board.name}${board.user_count ? ` <span class="chip-count">${board.user_count}</span>` : ''}`;
          chip.addEventListener('click', () => {
            this.querySelector('#boardName').value = board.name;
            this.querySelector('#board-form').dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }));
          });
          wrap.appendChild(chip);
        });
      }).catch(err => {
        if (refreshBtn) refreshBtn.classList.remove('spinning');
        const w = this.querySelector('#chips-wrap');
        if (w) w.innerHTML = '<span style="font-size:12px;color:rgba(255,255,255,0.3);">載入失敗，請點擊重新整理</span>';
        console.error('GetHotBoards failed:', err);
      });
    };

    // Initial load
    loadHotBoards();

    // Refresh button
    this.querySelector('#hot-boards-refresh-btn').addEventListener('click', () => {
      loadHotBoards();
    });

    this.querySelector('#board-form').addEventListener('submit', async (e) => {
      e.preventDefault();
      const loadingContainer = document.getElementById('loading-container');
      loadingContainer.style.display = 'flex';
      const boardName = this.querySelector('#boardName').value;
      const errorMessageDiv = this.querySelector('#error-message');

      try {
        const posts = await GotoBoard(boardName);
        errorMessageDiv.textContent = '';
        this.style.display = 'none';
        document.querySelector('post-list-page').style.display = 'block';
        const boardLabel = document.getElementById('board-name-label');
        if (boardLabel) boardLabel.textContent = boardName;
        setCurrentBoard(boardName);
        displayPosts(posts);
      } catch (error) {
        errorMessageDiv.textContent = error;
      } finally {
        loadingContainer.style.display = 'none';
      }
    });
  }
}

customElements.define('board-page', BoardPage);
