import {mkdir,writeFile,readdir} from 'node:fs/promises';
import path from 'node:path';
import {execFileSync} from 'node:child_process';

const pnpm=path.resolve('out/ci-pnpm',process.platform==='win32'?'pnpm.exe':'pnpm');
const version=execFileSync(pnpm,['--version'],{encoding:'utf8'}).trim();
if(version!=='12.22.0')throw Error('Unexpected native pnpm: '+version);
const archives=(await readdir('release')).filter(n=>n.endsWith('.tgz'));
if(archives.length!==1)throw Error('Expected exactly one release tarball');
const archive=path.resolve('release',archives[0]).split(path.sep).join('/');
const dir=path.resolve('out/install');
await mkdir(dir,{recursive:true});
await writeFile(path.join(dir,'package.json'),JSON.stringify({
 name:'companion-install-check',private:true,
 dependencies:{'@routilet/companion':'file:'+archive,'@routilet/c-exam-kit':'0.6.10'}
})+'\n');
if(process.platform==='win32')await writeFile(path.join(dir,'pnpm-workspace.yaml'),'nodeLinker: hoisted\npackageImportMethod: copy\nverifyStoreIntegrity: true\nenableGlobalVirtualStore: false\n');
const env=Object.fromEntries(Object.entries(process.env).filter(([k])=>!(/^(npm_config_|pnpm_|node_options$|node_path$)/i).test(k)));
for(const n of ['user.npmrc','global.npmrc'])await writeFile(path.join(dir,n),'');
Object.assign(env,{
 PNPM_HOME:path.join(dir,'pnpm-home'),
 NPM_CONFIG_USERCONFIG:path.join(dir,'user.npmrc'),
 NPM_CONFIG_GLOBALCONFIG:path.join(dir,'global.npmrc'),
 NPM_CONFIG_UPDATE_NOTIFIER:'false'
});
execFileSync(pnpm,['install','--prod','--ignore-scripts','--no-frozen-lockfile','--registry=https://registry.npmjs.org/','--store-dir='+path.join(dir,'store'),'--reporter=append-only'],{cwd:dir,env,stdio:'inherit',timeout:300000});
console.log('Installed candidate using the same fixed native pnpm and install flags as Routilet.');
