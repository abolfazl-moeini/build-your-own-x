import select, {Separator} from '@inquirer/select';
import {execSync} from 'child_process';

console.log('Welcome to the Framework Showcase!');

const answer = await select({
    message: 'Select the framework to showcase:',
    choices: [
        {
            name: 'SvelteKit',
            value: 'sveltekit',
            description: 'A framework for rapidly developing robust, performant web applications using Svelte',
        },
        {
            name: 'Vue.js',
            value: 'vue',
            description: "An MVVM front-end framework"
        },
        {
            name: 'Angular',
            value: 'angular',
            description: "Popular front-end framework for single-page apps and more"
        },
        new Separator(),
        {
            name: 'Express',
            value: 'express',
            description: "Back-end Node.js framework"
        },
        {
            name: 'Nest.js',
            value: 'nest2',
            description: "Progressive Node.js framework for building efficient, reliable and scalable server-side applications"
        },
        new Separator(),
        {
            name: 'Electron',
            value: 'electron',
            description: 'Framework for building desktop applications using web technologies',
        },
    ],
});


const dir = answer;

try {
    console.log(`Navigating into ${dir}...`);
    process.chdir(dir);
    console.log(`Current directory: ${process.cwd()}`);

    console.log('Installing dependencies...');
    execSync('npm install', {stdio: 'inherit'});  // show output in console

    console.log('Starting the project...');
    execSync('npm run dev', {stdio: 'inherit'});  // show output in console

} catch (err) {
    console.error(`Error: ${err}`);
}