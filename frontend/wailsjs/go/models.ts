export namespace main {
	
	export class AppMetrics {
	    heapAlloc: number;
	    sys: number;
	    numGC: number;
	    goroutines: number;
	    numCards: number;
	    numLists: number;
	    fileSizeMB: number;
	
	    static createFrom(source: any = {}) {
	        return new AppMetrics(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.heapAlloc = source["heapAlloc"];
	        this.sys = source["sys"];
	        this.numGC = source["numGC"];
	        this.goroutines = source["goroutines"];
	        this.numCards = source["numCards"];
	        this.numLists = source["numLists"];
	        this.fileSizeMB = source["fileSizeMB"];
	    }
	}

}

