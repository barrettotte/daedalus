export namespace daedalus {
	
	export class ListConfig {
	    title: string;
	    limit: number;
	
	    static createFrom(source: any = {}) {
	        return new ListConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.limit = source["limit"];
	    }
	}
	export class BoardConfig {
	    lists: Record<string, ListConfig>;
	    labelsExpanded?: boolean;
	    showYearProgress?: boolean;
	    collapsedLists?: string[];
	    halfCollapsedLists?: string[];
	
	    static createFrom(source: any = {}) {
	        return new BoardConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.lists = this.convertValues(source["lists"], ListConfig, true);
	        this.labelsExpanded = source["labelsExpanded"];
	        this.showYearProgress = source["showYearProgress"];
	        this.collapsedLists = source["collapsedLists"];
	        this.halfCollapsedLists = source["halfCollapsedLists"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CheckListItem {
	    idx: number;
	    desc: string;
	    done: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CheckListItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.idx = source["idx"];
	        this.desc = source["desc"];
	        this.done = source["done"];
	    }
	}
	export class Counter {
	    current: number;
	    max: number;
	    start: number;
	    step: number;
	    label: string;
	
	    static createFrom(source: any = {}) {
	        return new Counter(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.current = source["current"];
	        this.max = source["max"];
	        this.start = source["start"];
	        this.step = source["step"];
	        this.label = source["label"];
	    }
	}
	export class DateRange {
	    // Go type: time
	    start: any;
	    // Go type: time
	    end: any;
	
	    static createFrom(source: any = {}) {
	        return new DateRange(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.start = this.convertValues(source["start"], null);
	        this.end = this.convertValues(source["end"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CardMetadata {
	    id: number;
	    title: string;
	    // Go type: time
	    created?: any;
	    // Go type: time
	    updated?: any;
	    list_order: number;
	    // Go type: time
	    due?: any;
	    range?: DateRange;
	    labels: string[];
	    icon: string;
	    counter?: Counter;
	    checklist?: CheckListItem[];
	
	    static createFrom(source: any = {}) {
	        return new CardMetadata(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.created = this.convertValues(source["created"], null);
	        this.updated = this.convertValues(source["updated"], null);
	        this.list_order = source["list_order"];
	        this.due = this.convertValues(source["due"], null);
	        this.range = this.convertValues(source["range"], DateRange);
	        this.labels = source["labels"];
	        this.icon = source["icon"];
	        this.counter = this.convertValues(source["counter"], Counter);
	        this.checklist = this.convertValues(source["checklist"], CheckListItem);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	
	export class KanbanCard {
	    filePath: string;
	    listName: string;
	    metadata: CardMetadata;
	    previewText: string;
	
	    static createFrom(source: any = {}) {
	        return new KanbanCard(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.filePath = source["filePath"];
	        this.listName = source["listName"];
	        this.metadata = this.convertValues(source["metadata"], CardMetadata);
	        this.previewText = source["previewText"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace main {
	
	export class AppMetrics {
	    pid: number;
	    heapAlloc: number;
	    sys: number;
	    numGC: number;
	    goroutines: number;
	    numCards: number;
	    numLists: number;
	    maxID: number;
	    fileSizeMB: number;
	    processRSS: number;
	    processCPU: number;
	
	    static createFrom(source: any = {}) {
	        return new AppMetrics(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pid = source["pid"];
	        this.heapAlloc = source["heapAlloc"];
	        this.sys = source["sys"];
	        this.numGC = source["numGC"];
	        this.goroutines = source["goroutines"];
	        this.numCards = source["numCards"];
	        this.numLists = source["numLists"];
	        this.maxID = source["maxID"];
	        this.fileSizeMB = source["fileSizeMB"];
	        this.processRSS = source["processRSS"];
	        this.processCPU = source["processCPU"];
	    }
	}
	export class BoardResponse {
	    lists: Record<string, Array<daedalus.KanbanCard>>;
	    config?: daedalus.BoardConfig;
	    boardPath: string;
	
	    static createFrom(source: any = {}) {
	        return new BoardResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.lists = this.convertValues(source["lists"], Array<daedalus.KanbanCard>, true);
	        this.config = this.convertValues(source["config"], daedalus.BoardConfig);
	        this.boardPath = source["boardPath"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

