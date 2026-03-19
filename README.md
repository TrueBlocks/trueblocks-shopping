# Acrylic

Acrylic is a desktop app for painters who work from reference images and need help
translating them into real acrylic paint choices.

It lets you drop or paste an image, extract a dominant palette, match those colors
to paints in the local catalog, track which paints you already own, save favorite
mixes, and generate PDF exports for comparison or shopping.

## Features

- Create projects from dropped, pasted, or selected images
- Extract dominant colors from the source image
- Match colors to real paints and generated mixing recipes
- Track owned paints as a working inventory
- Save favorite color recipes
- Export comparison, detail, and shopping-list PDFs

## Data Storage

The app stores its data under `~/.local/share/trueblocks/acrylic/`.

Important files and folders:

- `acrylic.db` — SQLite database with paints, projects, matches, and favorites
- `state.json` — persisted UI state
- `projects/<id>/` — original images and thumbnails for each project

## Development

```bash
yarn install
yarn start
```

Useful commands:

- `yarn lint`
- `yarn type-check`
- `yarn test`
- `yarn build`

## Architecture Notes

- Backend: Go + Wails
- Frontend: React + TypeScript + Mantine
- Database: SQLite via `modernc.org/sqlite`
- Shared libraries: `packages/appkit`, `packages/color`, `@trueblocks/ui`, `@trueblocks/scaffold`
