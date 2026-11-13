# Changelog

## [0.7.4](https://github.com/hansehart/o3s/compare/v0.7.3...v0.7.4) (2026-11-13)


### Features

* **cli:** pin the stack's images by digest ([#233](https://github.com/hansehart/o3s/issues/233)) ([eb0dfff](https://github.com/hansehart/o3s/commit/eb0dfff1070fa8354bb291b5f2e6f2605723b05d))


### Bug Fixes

* **cli:** pull the gateway once per version ([#232](https://github.com/hansehart/o3s/issues/232)) ([f315bd2](https://github.com/hansehart/o3s/commit/f315bd2bc99340e5c9a8aebddabdc8ecd1e7bf33))

## [0.7.3](https://github.com/hansehart/o3s/compare/v0.7.2...v0.7.3) (2026-11-13)


### Features

* **cli:** keep the menu open in its own screen ([#230](https://github.com/hansehart/o3s/issues/230)) ([fbb2a7a](https://github.com/hansehart/o3s/commit/fbb2a7ab608819dc0381294f8e1d536e399d892e))

## [0.7.2](https://github.com/hansehart/o3s/compare/v0.7.1...v0.7.2) (2026-11-13)


### Features

* **cli:** add several features at once ([#228](https://github.com/hansehart/o3s/issues/228)) ([5a3b6b7](https://github.com/hansehart/o3s/commit/5a3b6b7be29fb4921fe6390233c9853bcc6ff42e))

## [0.7.1](https://github.com/hansehart/o3s/compare/v0.7.0...v0.7.1) (2026-11-13)


### Bug Fixes

* **repo:** install the devcontainer CLI on a bare home ([#226](https://github.com/hansehart/o3s/issues/226)) ([fbd4662](https://github.com/hansehart/o3s/commit/fbd466258688b282078189feadcd3a6334600047))

## [0.7.0](https://github.com/hansehart/o3s/compare/v0.6.6...v0.7.0) (2026-11-13)


### Features

* **gateway:** read the home's policy when it is mounted ([#183](https://github.com/hansehart/o3s/issues/183)) ([ba86b23](https://github.com/hansehart/o3s/commit/ba86b230b3f7f61e4176df9b76e1af01db4a6dbc))
* **repo:** allow, remove and list hosts with o3s firewall ([#182](https://github.com/hansehart/o3s/issues/182)) ([5d1ea5b](https://github.com/hansehart/o3s/commit/5d1ea5bb40eeb8fd10bf89eea0d42b62dbf18f7d))
* **repo:** bring a home up to date with o3s update ([#195](https://github.com/hansehart/o3s/issues/195)) ([6d7e883](https://github.com/hansehart/o3s/commit/6d7e883e385547e5b5ead4d32f620f9df6ae7dc8))
* **repo:** install o3s with curl and a checked script ([#204](https://github.com/hansehart/o3s/issues/204)) ([ee0067f](https://github.com/hansehart/o3s/commit/ee0067f54b274a6fbff8f093be1a32d6089b45d1))
* **repo:** keep what o3s makes in the XDG folders ([#209](https://github.com/hansehart/o3s/issues/209)) ([588d0e6](https://github.com/hansehart/o3s/commit/588d0e6ed637c2a45a37acb0f0ebc774e3a29d06))
* **repo:** lead through every dialog with a menu on bare o3s ([#196](https://github.com/hansehart/o3s/issues/196)) ([6e47924](https://github.com/hansehart/o3s/commit/6e47924d835edc85821c870dbea33a2e680e333d))
* **repo:** lead through the feature dialogs in the menu ([#199](https://github.com/hansehart/o3s/issues/199)) ([d91da69](https://github.com/hansehart/o3s/commit/d91da699c43f71786e58c43c9a2076b7224a9a3d))
* **repo:** make a folder into a home with o3s init ([#189](https://github.com/hansehart/o3s/issues/189)) ([50a2263](https://github.com/hansehart/o3s/commit/50a226374200aba6e71a3f3a3dcb15a1958d8056))
* **repo:** manage the cage's features with o3s feature ([#194](https://github.com/hansehart/o3s/issues/194)) ([3266c8d](https://github.com/hansehart/o3s/commit/3266c8d826cd693a65e3a21301ecdaff25c5cdd0))
* **repo:** manage the feature stores with o3s store ([#193](https://github.com/hansehart/o3s/issues/193)) ([ae25f84](https://github.com/hansehart/o3s/commit/ae25f8456d56249390e1afb24276b31bf7f740c3))
* **repo:** open the cage in VS Code or a shell ([#221](https://github.com/hansehart/o3s/issues/221)) ([5ff3793](https://github.com/hansehart/o3s/commit/5ff37938bc443b715e30eb54f7a47b3baa084d87))
* **repo:** prepare the host with o3s init on every start ([#208](https://github.com/hansehart/o3s/issues/208)) ([e2e7305](https://github.com/hansehart/o3s/commit/e2e73056c045795972140635aec65025509a831b))
* **repo:** publish the o3s binaries with every release ([#200](https://github.com/hansehart/o3s/issues/200)) ([c3e1b7c](https://github.com/hansehart/o3s/commit/c3e1b7cd7aa05ef665104a24feddc0aeb3c6d2af))
* **repo:** run the stack on the home's o3s.json ([#187](https://github.com/hansehart/o3s/issues/187)) ([80d38ea](https://github.com/hansehart/o3s/commit/80d38eaefd757c8c1093d67832f0f34a0271875b))
* **repo:** ship the schema of o3s.json with every home ([#191](https://github.com/hansehart/o3s/issues/191)) ([22981ef](https://github.com/hansehart/o3s/commit/22981efb5869ce42eb88250d391c83cd204b0ac0))
* **repo:** store, remove and list secrets with o3s secret ([#188](https://github.com/hansehart/o3s/issues/188)) ([0429977](https://github.com/hansehart/o3s/commit/042997728bae16d2bb90b1b27fb4616782a5819d))
* **repo:** update the binary and rebuild the cage with o3s update ([#205](https://github.com/hansehart/o3s/issues/205)) ([ba9e383](https://github.com/hansehart/o3s/commit/ba9e383bc254e7d231f828fd1ec6d2f4ac4d2325))


### Bug Fixes

* **gateway:** start on an empty policy ([#179](https://github.com/hansehart/o3s/issues/179)) ([0ceedb5](https://github.com/hansehart/o3s/commit/0ceedb57da0f051149c50e316225b725bd4cd713))
* **repo:** hold every entry of o3s.json to its rules on load ([#192](https://github.com/hansehart/o3s/issues/192)) ([9afa168](https://github.com/hansehart/o3s/commit/9afa168602dabe8860d7d1a9f32018c2d8a59ed9))
* **repo:** keep the host's secrets out of a home's cage ([#212](https://github.com/hansehart/o3s/issues/212)) ([bb58afa](https://github.com/hansehart/o3s/commit/bb58afa20f850a7848ba3453dbdb709f6861d57d))
* **repo:** point a new home at o3s cage ide ([#224](https://github.com/hansehart/o3s/issues/224)) ([f58bc5e](https://github.com/hansehart/o3s/commit/f58bc5eef5093018d0786121e652755ba862bfb6))
* **repo:** smooth the menu's output and the host rule ([#213](https://github.com/hansehart/o3s/issues/213)) ([37b95aa](https://github.com/hansehart/o3s/commit/37b95aae1c9c9e891b732b60887a571fadb00eec))


### Reverts

* **repo:** take the release scope back out ([c0678c3](https://github.com/hansehart/o3s/commit/c0678c39da42ff65702b4c20702ab1db3980b94a))
