# Kanban

> Kanban app in (pure) Go using the Gio toolkit.

This application implements a simple Kanban workflow.
It represents a small vertical slice of what a desktop app might look like with
Gio, in both structure and aesthetic.

The codebase is prototypical and contains some experiments that may fall outside
Gio idioms. In that light, this could make a great place for discovering patterns
that work well with Gio for the desktop environment.

- Data is stored in a [Bolt](https://github.com/etcd-io/bbolt) database on disk.
- GUI is rendered via [Gio](https://gioui.org/).

Feedback welcome!

`go get -u git.sr.ht/~jackmordaunt/kanban/... && kanban`

![main-view](https://git.sr.ht/~jackmordaunt/kanban/blob/master/img/main-view.png)
![edit-view](https://git.sr.ht/~jackmordaunt/kanban/blob/master/img/edit-view.png)
