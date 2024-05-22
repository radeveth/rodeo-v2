# Rodeo V2 (Alamo)

Rodeo is leverage yield farming protocol with it's own lending pool.

## Starting

Use `make` to run the app, it should watch for changes and restart.

The app should now be accessible from `http://localhost:8000`.

You will need a Golang toolchain installed.

The app has some useful "jobs" that can be run from the CLI, scheduled or dispatched in a queue.

You can see a list of them using `make run help`.

One command you will need to ran initially would be `make run db-migrate-up` ;)

## Javascript

As much as we would like to avoid the compilation step and just write vanilla JS with a few imported modules, re-implementing wallet connect without their libraries is a larger project for the next bear market.

So right now the little JS code we need to handle wallet connections and sending transactions lives in `support/app/app.js`.

To build it and output it to `assets/app.js` run `npm run build` in the `support/app/` folder.

## Contracts

Contracts (in the `contracts/`) folder are built and deployed using [Foundry](https://book.getfoundry.sh/) with some common commands specified in the `commands/makefile`.

We initially were trying to write contracts in the style of DAI's codebase but have since relaxed it a bit: we use long enough variable names to describe things, not just 3 letter, but still aim to be concise.

We've kept many of the other goals / lessons:

- Avoid inheritance
- Avoid imports
- Avoid modifiers (mostly)
- Avoid calling too many internal methods
- Avoid proxies and upgradability
- Avoid too complex storage/slot manipulation
- Optimize for readability
- Optimize for concisness
- Do compose multiple contracts together in order to build a more complex system

The general goal is to be able to read a contract top to bottom and understand everything that's going on, no jumping left and right and adding up all kinds of small pieces in your head.

## Usefule snippets

Recalculate leaderboard points

```sql
update leaderboards_users set points = coalesce((select sum(points) from leaderboards_points where user_id = leaderboards_users.id), 0);
```
