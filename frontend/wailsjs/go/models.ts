export namespace kfiles {
	
	export class FileBrowser {
	    PlatformPath: string;
	    KubeLoadsDir: string;
	    KubeRestsDir: string;
	    DbPath: string;
	    NodeRegistry?: kubenet.NodeRegistry;
	
	    static createFrom(source: any = {}) {
	        return new FileBrowser(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.PlatformPath = source["PlatformPath"];
	        this.KubeLoadsDir = source["KubeLoadsDir"];
	        this.KubeRestsDir = source["KubeRestsDir"];
	        this.DbPath = source["DbPath"];
	        this.NodeRegistry = this.convertValues(source["NodeRegistry"], kubenet.NodeRegistry);
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
	export class FileType {
	    id: string;
	    name: string;
	    type: string;
	    extension?: string;
	    path: string;
	    size?: string;
	    sizeInBytes?: number;
	    lastModified: string;
	    // Go type: time
	    lastModifiedDate: any;
	    isRefrigerated?: boolean;
	    compressionRatio?: number;
	    owner: string;
	    isShared: boolean;
	    sharedWith?: string[];
	    itemCount?: number;
	    isDistributed?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new FileType(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.type = source["type"];
	        this.extension = source["extension"];
	        this.path = source["path"];
	        this.size = source["size"];
	        this.sizeInBytes = source["sizeInBytes"];
	        this.lastModified = source["lastModified"];
	        this.lastModifiedDate = this.convertValues(source["lastModifiedDate"], null);
	        this.isRefrigerated = source["isRefrigerated"];
	        this.compressionRatio = source["compressionRatio"];
	        this.owner = source["owner"];
	        this.isShared = source["isShared"];
	        this.sharedWith = source["sharedWith"];
	        this.itemCount = source["itemCount"];
	        this.isDistributed = source["isDistributed"];
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
	export class SearchResult {
	    file: FileType;
	    score: number;
	    match_type: string;
	    excerpt: string;
	
	    static createFrom(source: any = {}) {
	        return new SearchResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.file = this.convertValues(source["file"], FileType);
	        this.score = source["score"];
	        this.match_type = source["match_type"];
	        this.excerpt = source["excerpt"];
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
	export class StorageInfo {
	    total_used: number;
	    total_used_mb: number;
	    total_used_gb: number;
	    percentage_used: number;
	    total_capacity: number;
	    remaining_bytes: number;
	    remaining_gb: number;
	
	    static createFrom(source: any = {}) {
	        return new StorageInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total_used = source["total_used"];
	        this.total_used_mb = source["total_used_mb"];
	        this.total_used_gb = source["total_used_gb"];
	        this.percentage_used = source["percentage_used"];
	        this.total_capacity = source["total_capacity"];
	        this.remaining_bytes = source["remaining_bytes"];
	        this.remaining_gb = source["remaining_gb"];
	    }
	}

}

export namespace kubenet {
	
	export class Node {
	    ID: string;
	    IP: string;
	    Hostname: string;
	    // Go type: time
	    LastSeen: any;
	    Version: number;
	    Broadcasted: boolean;
	    Status: string;
	
	    static createFrom(source: any = {}) {
	        return new Node(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.IP = source["IP"];
	        this.Hostname = source["Hostname"];
	        this.LastSeen = this.convertValues(source["LastSeen"], null);
	        this.Version = source["Version"];
	        this.Broadcasted = source["Broadcasted"];
	        this.Status = source["Status"];
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
	export class NodeRegistry {
	
	
	    static createFrom(source: any = {}) {
	        return new NodeRegistry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}

}

