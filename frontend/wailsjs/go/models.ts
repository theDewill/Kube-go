export namespace kfiles {
	
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

}

