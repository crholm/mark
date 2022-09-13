# Mark, a small note-taking app

Mark is a note-taking app for the terminal supporting editor and pager of choose along with **markdown highlighting**, **tags**, **fulltext search** with built in index and storage through git 
. Utilizing  

### Installation 
```bash 
go install github.com/crholm/mark/cmd/mark@latest
```

### Example usage

**Taking a note**
```bash 
## Opens editor to create a note on save and exit (uses whats in env EDITOR or nano)
$ mark

$ mark The title -- and the content of the note

$ mark or just the content

$ mark With a tags -- "add #tags using the pound sign #example"

$ mark With more tags -- "tags linking documents #example"


$ mark Markdown Example -- "
# Markdown syntax guide

## Headers

# This is a Heading h1
## This is a Heading h2 
###### This is a Heading h6

## Emphasis

*This text will be italic*  
_This will also be italic_

**This text will be bold**  
__This will also be bold__

_You **can** combine them_

## Lists

### Unordered

* Item 1
* Item 2
* Item 2a
* Item 2b
"
```

**Listing notes**
```bash 
$ mark ls                                                                                                 
2022-08-12_14:04:49Z_Friday.md
2022-08-12_14:04:52Z_Friday.md
2022-08-12_14:05:08Z_Friday.md
2022-08-12_14:05:14Z_Friday.md

## Lists title and tags
$ mark ll
2022-08-12_15:01:49Z_Friday.md       TODO [todo]
2022-08-18_08:04:08Z_Thursday.md     Links
2022-08-18_09:11:42Z_Thursday.md     


## Prefix matching 
$ mark ls 2022-08-12_14:04
2022-08-12_14:04:49Z_Friday.md
2022-08-12_14:04:52Z_Friday.md

## Searching for tag 
$ mark ls :example # or mark ls "#example"
2022-08-12_14:05:08Z_Friday.md

## Full text seach 
$ mark ls heading
2022-08-12_14:05:14Z_Friday.md

```

**Print notes in terminal**
```bash
$ mark cat 
## Prints all notes with markdown syntax highlighting 

$ mark cat :example
## Prints all notes containing the tag #example

## Print a specific note
$ mark cat 2022-08-12_14:04:52Z_Friday
```

**Look at notes using a pager**
```bash
## Opens a pager with all notes (uses whats in env PAGER or less) 
$ mark pager 

## Opens a pager with all notes containing tag #example (uses whats in env PAGER or less)
$ mark pager :example
## Prints all notes containing the tag #example

## Opens a pager for a specific note (uses whats in env PAGER or less)
$ mark pager 2022-08-12_14:04:52Z_Friday

## Opens a pager all notes in August of 2022 (uses whats in env PAGER or less)
$ mark pager 2022-08
```


**Edit a note**
```bash
## Opens editor for specific note (uses whats in env EDITOR or nano) 
$ mark edit 2022-08-12_14:04:52Z_Friday

## Opens editor for note of choice from a tag (uses whats in env EDITOR or nano) 
$ mark edit :example
1 - 2022-08-12_14:05:08Z_Friday
2 - 2022-08-12_14:25:04Z_Friday
? 2 <Enter>  # Opens file 2 in editor 

## Opens editor for note of choice from full text search (uses whats in env EDITOR or nano) 
$ mark edit content
1 - 2022-08-12_14:04:49Z_Friday
2 - 2022-08-12_14:04:52Z_Friday
? 1 <Enter>  # Opens file 1 in editor 
```

**Remove a note**
```bash
## Removes one particular note
$ mark rm 2022-08-12_14:04:52Z_Friday
```


**Note selector / Picker / FZF**
```bash
$ export MARK_PICKER="fzf --preview \"mark cat {1}\""
$ mark edit
>2022-08-12_15:01:49Z_Friday.md       TODO [todo]           ┌──────────────────────────────────────────
 2022-08-18_08:04:08Z_Thursday.md     Links                 │ ┌─   TODO ─────────── Friday Aug 12 2022                                                                      
 2/2                                                        │ │    ## Done                             
```

You can also use a custom "picker" in a grep fashion. eg.

```bash
$ export MARK_PICKER="fzf"
$ mark --grep edit
  2022-08-18_09:11:42Z_Thursday.md:2: title: "A note"                                                                                                                                                                                                         
  2022-08-18_09:11:42Z_Thursday.md:1: ---                                                                                                                                                                                                                        
  2022-09-13_12:57:19Z_Tuesday.md:10:                                                                                                                                                                                                                            
  2022-09-13_12:57:19Z_Tuesday.md:9:                                                                                                                                                                                                                             
  2022-09-13_12:57:19Z_Tuesday.md:8: cut                                                                                                                                                                                                                         
  2022-09-13_12:57:19Z_Tuesday.md:7: ---                                                                                                                                                                                                                         
  2022-09-13_12:57:19Z_Tuesday.md:6: updated_at: 2022-09-13T15:51:42.769978695+02:00                                                                                                                                                                             
  2022-09-13_12:57:19Z_Tuesday.md:5: created_at: 2022-09-13T14:57:19.333373569+02:00                                                                                                                                                                             
  2022-09-13_12:57:19Z_Tuesday.md:4: tags: ["example"]                                                                                                                                                                                                                    
  2022-09-13_12:57:19Z_Tuesday.md:3: alias: ""                                                                                                                                                                                                                   
  2022-09-13_12:57:19Z_Tuesday.md:2: title: "A test note"                                                                                                                                                                                                                   
> 2022-09-13_12:57:19Z_Tuesday.md:1: ---                                                                                                                                                                                                                         
  96/96            
  
## Or export MARK_PICKER_MODE="grep" can be used instead of flag --grep                                       
```


**git stuff**
```bash
## Using git straight up for versioning
$ mark git <any git command with dir set to storage>

$ mark git init . 
$ mark git add .
$ mark git commit -m "init"
$ mark git remote add <name> <url> 
$ mark git push <name>

```


**Sync notes with repo**
```bash
## sync notes with your git repo 
$ mark sync
## Short hand for  
$ mark git add .
$ mark git commit -m "sync commit"
$ mark git pull
$ mark git push
```

**Recalculate full text search and tag index**
```bash 
$ mark reindex
```

