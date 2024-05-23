import {GotoBoard} from '../../wailsjs/go/pttclient/PttClient.js';
import {displayPosts} from './post.js';

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
          </div>
        `;

        this.querySelector('#boardName').focus();
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
                // posts.sort(function(a,b) {
                //     return b.search_id - a.search_id
                // });
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
