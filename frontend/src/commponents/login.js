import logo from '../assets/images/logo-universal.png';
import live from '../assets/images/live.png';

import {Login} from '../../wailsjs/go/pttclient/PttClient.js';

class LoginPage extends HTMLElement {
    connectedCallback() {
        this.innerHTML = `
            <div class="page">
                <img id="logo" class="logo">
                <form id="login-form">
                    <div id="error-message" style="color: red;"></div>
                    <div class="input-box">
                        <label for="account">帳號</label>
                        <input class="input" id="account" type="text" autocapitalize="none" autocomplete="off" />
                    </div>
                    <div class="input-box">
                        <label for="password">密碼</label>
                        <input class="input" id="password" type="password" />
                    </div>
                    <div class="input-box">
                        <button class="login-btn btn" type="submit">登入</button>
                    </div>
                </form>
            </div>
`;

        this.querySelector('#logo').src = live;

        this.querySelector('#account').focus();
        this.querySelector('#login-form').addEventListener('submit', (e) => {
            e.preventDefault();
            const loadingContainer = document.getElementById('loading-container');
            loadingContainer.style.display = 'flex';
            const account = this.querySelector('#account').value;
            const password = this.querySelector('#password').value;
            const errorMessageDiv = this.querySelector('#error-message');

            Login(account, password)
                .then((result) => {
                    errorMessageDiv.textContent = '';
                    this.style.display = 'none';
                    document.querySelector('board-page').style.display = 'block';
                })
                .catch((err) => {
                    console.error(err);
                    errorMessageDiv.textContent = err;
                })
                .finally(() => {
                    loadingContainer.style.display = 'none';
                });
        });
    }
}

customElements.define('login-page', LoginPage);