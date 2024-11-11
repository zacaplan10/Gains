import './style.css';
import './app.css';

import logo from './assets/images/STGLogo.jpg';
import {GetCapitalGainsBalance} from '../wailsjs/go/main/App';

document.querySelector('#app').innerHTML = `
    <img id="logo" class="logo">
      <div class="result" id="result">Enter Schwab account id below 👇</div>
      <div class="input-box" id="input">
        <input class="input" id="name" type="text" autocomplete="off" />
        <button class="btn" onclick="getCapitalGainsBalance()">Calculate</button>
      </div>
    </div>
`;
document.getElementById('logo').src = logo;

let nameElement = document.getElementById("name");
nameElement.focus();
let resultElement = document.getElementById("result");

// Setup the getCapitalGainsBalance function
window.getCapitalGainsBalance = function () {
    // Get name
    let name = nameElement.value;

    // Check if the input is empty
    if (name === "") return;

    // Call App.GetCapitalGainsBalance(name)
    try {
        GetCapitalGainsBalance(name)
            .then((result) => {
                // Update result with data back from App.GetCapitalGainsBalance()
                resultElement.innerText = result;
            })
            .catch((err) => {
                console.error(err);
            });
    } catch (err) {
        console.error(err);
    }
};
