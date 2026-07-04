# YVCDB — Your Vibe Code Deserves Better

[![CI](https://github.com/Morialkar/yvcdb/actions/workflows/ci.yml/badge.svg)](https://github.com/Morialkar/yvcdb/actions/workflows/ci.yml)
[![Release](https://github.com/Morialkar/yvcdb/actions/workflows/release.yml/badge.svg)](https://github.com/Morialkar/yvcdb/releases)
[![Coverage](https://raw.githubusercontent.com/Morialkar/yvcdb/badges/coverage.svg)](https://github.com/Morialkar/yvcdb/actions/workflows/ci.yml)

*[English documentation](README.md)*

YVCDB est un CLI interactif qui orchestre le refactoring structuré d'une codebase via Claude Code ou Codex CLI. Il exécute des phases de revue spécialisées, isole les changements parallèles dans des worktrees Git, et garde une étape d'approbation humaine avant d'intégrer les changements.

L'interface est en anglais par défaut et supporte aussi le français.

## La méthodologie AFTER

YVCDB est l'implémentation de référence de la moitié « Test Everything Rigorously » de la méthodologie AFTER (Architect First, Test Everything Rigorously), mon approche personnelle du développement assisté par IA, appliquée aux codebases existantes générées par IA. Les deux moitiés correspondent aux deux extrémités du workflow :

- **Architect First** : aucune génération de code avant la spécification. Les specs détaillées, les fichiers de standards de qualité (`CLAUDE.md`), les contraintes explicites et les décisions d'architecture sont pris par l'humain avant que l'IA ne génère quoi que ce soit.
- **Test Everything Rigorously** : rien n'est livré sur la confiance. Le code généré passe par des tests (cas nominaux, limites et erreurs), une revue par phases avec des points de validation humaine, et une checklist finale approuvée par l'humain. La valeur de l'ingénieure apparaît après la génération, d'où le nom.

## Prérequis

- Go 1.26 ou plus récent
- [Claude Code CLI](https://docs.anthropic.com/en/docs/claude-code) ou Codex CLI
- Git, sauf si YVCDB est lancé avec `--no-git`
- Une session authentifiée pour le fournisseur choisi

Vérifiez Go, Git et le fournisseur que vous comptez utiliser :

```sh
go version
git --version

# L'un des deux est requis :
claude --version
codex --version
```

## Installation

### Release précompilée

Téléchargez l'archive correspondant à votre système et architecture depuis la [dernière release GitHub](https://github.com/Morialkar/yvcdb/releases/latest). Chaque archive contient la commande `yvcdb` ainsi que l'alias rétrocompatible `tvcmm`.

- macOS et Linux : extrayez l'archive `.tar.gz` et déplacez `yvcdb` dans un répertoire de votre `PATH`, comme `/usr/local/bin`.
- Windows : extrayez l'archive `.zip` et ajoutez son répertoire à votre `PATH`.

Vérifiez les fichiers téléchargés avec le `checksums.txt` de la même release.

Confirmez la version installée avec `yvcdb --version`.

### Installation avec Go

Directement depuis le proxy de modules :

```sh
go install github.com/Morialkar/yvcdb@latest
```

Ou depuis un clone local :

```sh
go install .
```

Ceci installe la commande principale `yvcdb` dans `$(go env GOPATH)/bin`.

Pour installer aussi l'alias rétrocompatible `tvcmm` :

```sh
go install ./...
```

Assurez-vous que le répertoire des binaires Go est dans votre `PATH` :

```sh
export PATH="$(go env GOPATH)/bin:$PATH"
```

## Configuration

Lancez l'outil de configuration interactif une fois :

```sh
yvcdb config
```

Il configure :

- la langue de l'interface et des réponses : `en` ou `fr` ;
- le fournisseur de CLI IA : `claude` ou `codex` ;
- le modèle par défaut du fournisseur, comme `sonnet` pour Claude ou `gpt-5.4` pour Codex.

Les défauts sont anglais, Claude et `sonnet`. Sur macOS, la configuration est stockée dans :

```text
~/Library/Application Support/yvcdb/config.json
```

YVCDB lit la configuration legacy `tvcmm` si aucune configuration YVCDB n'existe.

Le fichier persistant peut aussi être édité directement :

```json
{
  "language": "fr",
  "provider": "codex",
  "model": "gpt-5.4"
}
```

Codex s'exécute en mode non interactif avec sortie JSONL, sessions éphémères et sandbox `workspace-write`. Claude utilise son mode de sortie `stream-json`.

YVCDB embarque des jeux de prompts parallèles en anglais et en français. La langue configurée sélectionne à la fois les chaînes d'interface et le jeu de prompts embarqué.

## Utilisation

Lancez YVCDB sur le répertoire courant :

```sh
yvcdb
```

Ou spécifiez un projet :

```sh
yvcdb /chemin/vers/projet
```

Surcharges courantes :

```sh
yvcdb --model opus --lang fr --max-turns 30 /chemin/vers/projet
yvcdb --provider codex --model gpt-5.4 /chemin/vers/projet
yvcdb --phase security /chemin/vers/projet
yvcdb --no-git /chemin/vers/projet
```

Drapeaux disponibles :

| Drapeau | Description |
| --- | --- |
| `--provider claude\|codex` | Surcharge le fournisseur de CLI IA configuré pour cette exécution |
| `--model <modèle>` | Surcharge le modèle configuré pour cette exécution |
| `--lang en\|fr` | Surcharge la langue configurée pour cette exécution |
| `--max-turns <n>` | Nombre maximum de tours pour Claude ; défaut : `20`. Codex CLI n'a pas d'équivalent |
| `--phase <id>` | Démarre à `diagnostic`, `safety`, `security`, `structure`, `readability` ou `devil` |
| `--no-git` | Désactive branches, commits, worktrees et merges automatiques |

Le modèle sélectionné est toujours affiché pour confirmation avant le démarrage du pipeline.

## Pipeline de refactoring

YVCDB exécute quatre étapes :

1. **Diagnostic** — inventorie la codebase et identifie les risques sans modifier de fichiers.
2. **Filet de sécurité** — ajoute des tests smoke et enregistre l'état courant.
3. **Revue parallèle** — exécute les phases sécurité, structure et lisibilité dans des worktrees Git séparés.
4. **Avocat du diable** — effectue une revue finale sans ménagement.

Chaque phase complétée attend une décision humaine :

| Touche | Action |
| --- | --- |
| `y` ou `o` | Approuve et commit le résultat |
| `r` | Réitère avec le contexte de l'itération précédente |
| `f` | Envoie un retour libre et précis à l'agent et relance une itération |
| `s` | Skip le résultat |
| `q` | Quitte et annule les sous-processus d'agent actifs |

Pendant l'étape parallèle, utilisez `Tab` ou `1`–`3` pour naviguer entre les exécutions.

Après toutes les phases, YVCDB présente une checklist de qualité en huit points. Les critères échoués peuvent passer par une boucle de correction interactive supplémentaire.

## Comportement Git

Quand l'intégration Git est active, YVCDB :

- propose d'initialiser un dépôt s'il n'en existe pas ;
- crée des branches de phase nommées `refactor/<timestamp>/<phase>` ;
- exécute les phases parallèles dans des worktrees temporaires sous le répertoire temporaire système ;
- commit les changements approuvés ;
- rebase les branches parallèles séquentiellement sur la branche de base mise à jour ;
- les intègre avec des merges fast-forward.

Si une création de branche, un commit, un rebase ou un merge échoue, YVCDB arrête ce chemin et signale l'erreur au lieu d'avancer silencieusement. Les rebases en conflit sont annulés et leurs worktrees préservés pour une résolution manuelle. Lancez l'outil avec un arbre de travail propre pour des résultats prévisibles.

## Logs

Les événements bruts du fournisseur sont écrits dans :

```text
<projet>/refactor-logs/<timestamp>_<phase>_iter<n>.md
```

Le répertoire est ignoré par le `.gitignore` de ce dépôt.

## Développement

```sh
go test ./...
go vet ./...
go build ./...
```

La CI exécute ces vérifications à chaque push et pull request, compile nativement sur Linux, macOS et Windows, et rejette une couverture totale sous 93 %. Les packages `main` de point d'entrée sont exclus de la mesure. Le badge de couverture est généré par la CI elle-même et poussé sur la branche `badges` — aucun service externe impliqué.

Pour publier une release, poussez un tag de version sémantique :

```sh
git tag v1.0.0
git push origin v1.0.0
```

Le workflow de release relance la CI, construit les archives `amd64` et `arm64` pour macOS, Linux et Windows, publie les checksums et les notes de release, et crée les attestations d'artefacts GitHub.

Les prompts de phase localisés sont embarqués depuis `cmd/prompts/en/` et `cmd/prompts/fr/`. L'orchestration principale vit dans `internal/tui`, l'exécution des fournisseurs dans `internal/runner`, et les opérations Git dans `internal/git`.

## Comment cet outil a été construit

YVCDB a été développé avec deux assistants IA — Claude et Codex — selon la méthodologie AFTER décrite plus haut. L'humain conçoit l'architecture et le workflow en amont (phases, stratégie Git, boucles d'approbation), les assistants implémentent selon ce design, puis chaque comportement est verrouillé par des tests rigoureux — incluant les chemins d'erreur, les courses et les deadlocks, dont plusieurs ont été attrapés par les tests eux-mêmes. Un article détaillant la méthodologie arrive bientôt.

## Licence

YVCDB est distribué sous la [licence MIT](LICENSE).
