import { useState, useEffect } from "react";
import {
  Search,
  FileText,
  Download,
  ExternalLink,
  Brain,
  Sparkles,
  Filter,
  SortAsc,
  SortDesc,
  AlertCircle,
  Clock,
  Zap,
  Settings,
} from "lucide-react";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { AppLayout } from "@/components/layout/AppLayout";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";
import { FileBrowserAPI, SearchResult } from "@/lib/file-api";
import { fileIconMap } from "@/lib/file-icons";
import { File as FileType } from "@/types/file-types";
import { toast } from "sonner";

interface FinderProps {
  onFileSelect?: (file: FileType) => void;
}

export default function Finder({ onFileSelect }: FinderProps) {
  const [query, setQuery] = useState("");
  const [results, setResults] = useState<SearchResult[]>([]);
  const [isSearching, setIsSearching] = useState(false);
  const [searchHistory, setSearchHistory] = useState<string[]>([]);
  const [sortBy, setSortBy] = useState<"relevance" | "date" | "name">("relevance");
  const [sortOrder, setSortOrder] = useState<"asc" | "desc">("desc");
  const [hasSearched, setHasSearched] = useState(false);
  const [searchTime, setSearchTime] = useState<number>(0);
  const [selectedModel, setSelectedModel] = useState("ollama");
  const [settingsOpen, setSettingsOpen] = useState(false);

  // Load search history and preferences from localStorage on mount
  useEffect(() => {
    const stored = localStorage.getItem("finder-search-history");
    if (stored) {
      setSearchHistory(JSON.parse(stored));
    }

    const storedModel = localStorage.getItem("finder-preferred-model");
    if (storedModel) {
      setSelectedModel(storedModel);
    }
  }, []);

  // Save search history to localStorage
  useEffect(() => {
    localStorage.setItem("finder-search-history", JSON.stringify(searchHistory));
  }, [searchHistory]);

  // Save preferred model to localStorage
  useEffect(() => {
    localStorage.setItem("finder-preferred-model", selectedModel);
  }, [selectedModel]);

  const handleSearch = async () => {
    if (!query.trim()) return;

    setIsSearching(true);
    setHasSearched(true);
    const startTime = Date.now();

    try {
      // Add to search history
      const newHistory = [query, ...searchHistory.filter((h) => h !== query)].slice(0, 5);
      setSearchHistory(newHistory);

      // Perform search
      const searchResults = await FileBrowserAPI.searchFiles(query, 20);
      setResults(searchResults || []);

      const endTime = Date.now();
      setSearchTime(endTime - startTime);
    } catch (error) {
      console.error("Search failed:", error);
      setResults([]);
    } finally {
      setIsSearching(false);
    }
  };

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === "Enter") {
      handleSearch();
    }
  };

  const sortResults = (results: SearchResult[]) => {
    const sorted = [...results].sort((a, b) => {
      let comparison = 0;

      switch (sortBy) {
        case "relevance":
          comparison = b.score - a.score;
          break;
        case "date":
          comparison =
            new Date(b.file.lastModifiedDate).getTime() -
            new Date(a.file.lastModifiedDate).getTime();
          break;
        case "name":
          comparison = a.file.name.localeCompare(b.file.name);
          break;
      }

      return sortOrder === "asc" ? -comparison : comparison;
    });

    return sorted;
  };

  const getMatchTypeIcon = (matchType: string) => {
    switch (matchType) {
      case "exact":
        return <Zap className="h-4 w-4 text-green-500" />;
      case "semantic":
        return <Brain className="h-4 w-4 text-blue-500" />;
      default:
        return <Search className="h-4 w-4 text-gray-500" />;
    }
  };

  const getMatchTypeLabel = (matchType: string) => {
    switch (matchType) {
      case "exact":
        return "Exact match";
      case "semantic":
        return "Semantic match";
      default:
        return "Partial match";
    }
  };

  const getModelBadgeColor = (model: string) => {
    switch (model) {
      case "ollama":
        return "bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200";
      case "gemini":
        return "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200";
      default:
        return "bg-gray-100 text-gray-800 dark:bg-gray-900 dark:text-gray-200";
    }
  };

  const getModelDisplayName = (model: string) => {
    switch (model) {
      case "ollama":
        return "Ollama";
      case "gemini":
        return "Gemini";
      default:
        return "Simple";
    }
  };

  const handleDownload = async (file: FileType) => {
    try {
      const blob = await FileBrowserAPI.downloadFile(file.path);
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = file.name;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);
    } catch (error) {
      console.error("Download failed:", error);
    }
  };

  const highlightMatch = (text: string, query: string) => {
    if (!query) return text;

    const regex = new RegExp(`(${query})`, "gi");
    const parts = text.split(regex);

    return parts.map((part, index) =>
      regex.test(part) ? (
        <mark key={index} className="bg-yellow-200 dark:bg-yellow-800 rounded px-1">
          {part}
        </mark>
      ) : (
        part
      ),
    );
  };

  const sortedResults = sortResults(results || []);

  return (
    <AppLayout>
      <div className="flex flex-col h-full p-6 space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div className="flex items-center space-x-2">
            <Brain className="h-6 w-6 text-primary" />
            <h1 className="text-2xl font-bold">Smart Finder</h1>
            <Badge variant="secondary" className="ml-2">
              <Sparkles className="h-3 w-3 mr-1" />
              AI-Powered
            </Badge>
            <Badge className={getModelBadgeColor(selectedModel)}>
              {getModelDisplayName(selectedModel)}
            </Badge>
          </div>

          <Dialog open={settingsOpen} onOpenChange={setSettingsOpen}>
            <DialogTrigger asChild>
              <Button variant="outline" size="sm">
                <Settings className="h-4 w-4 mr-2" />
                Settings
              </Button>
            </DialogTrigger>
            <DialogContent className="sm:max-w-[425px]">
              <DialogHeader>
                <DialogTitle>Finder Settings</DialogTitle>
                <DialogDescription>
                  Configure your AI model preferences for file analysis and search.
                </DialogDescription>
              </DialogHeader>
              <div className="grid gap-4 py-4">
                <div className="grid gap-2">
                  <Label htmlFor="search-model">AI Model for Search Analysis</Label>
                  <Select value={selectedModel} onValueChange={setSelectedModel}>
                    <SelectTrigger id="search-model">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="ollama">
                        <div className="flex items-center space-x-2">
                          <div className="w-2 h-2 bg-blue-500 rounded-full"></div>
                          <span>Ollama (Local AI)</span>
                        </div>
                      </SelectItem>
                      <SelectItem value="gemini">
                        <div className="flex items-center space-x-2">
                          <div className="w-2 h-2 bg-green-500 rounded-full"></div>
                          <span>Google Gemini (Cloud AI)</span>
                        </div>
                      </SelectItem>
                      <SelectItem value="simple">
                        <div className="flex items-center space-x-2">
                          <div className="w-2 h-2 bg-gray-500 rounded-full"></div>
                          <span>Simple Analysis (No AI)</span>
                        </div>
                      </SelectItem>
                    </SelectContent>
                  </Select>
                  <div className="text-xs text-muted-foreground">
                    {selectedModel === "ollama" &&
                      "Uses your local Ollama installation. Best for privacy and offline use."}
                    {selectedModel === "gemini" &&
                      "Uses Google Gemini API. Requires internet and API key. Most advanced analysis."}
                    {selectedModel === "simple" &&
                      "Basic keyword matching without AI. Fastest but less accurate."}
                  </div>
                </div>
              </div>
              <DialogFooter>
                <Button onClick={() => setSettingsOpen(false)}>Save Settings</Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
        </div>

        {/* Search Interface */}
        <Card>
          <CardHeader>
            <CardTitle className="text-lg">Describe what you're looking for</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex space-x-2">
              <div className="relative flex-1">
                <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                <Input
                  value={query}
                  onChange={(e) => setQuery(e.target.value)}
                  onKeyPress={handleKeyPress}
                  placeholder="e.g., 'my presentation about quarterly results' or 'document with budget data'"
                  className="pl-10"
                  disabled={isSearching}
                />
              </div>
              <Button onClick={handleSearch} disabled={isSearching || !query.trim()}>
                {isSearching ? (
                  <>
                    <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-current mr-2" />
                    Searching...
                  </>
                ) : (
                  <>
                    <Search className="h-4 w-4 mr-2" />
                    Search
                  </>
                )}
              </Button>
            </div>

            {/* Search History */}
            {searchHistory.length > 0 && (
              <div className="space-y-2">
                <p className="text-sm text-muted-foreground">Recent searches:</p>
                <div className="flex flex-wrap gap-2">
                  {searchHistory.map((historyQuery, index) => (
                    <Badge
                      key={index}
                      variant="outline"
                      className="cursor-pointer hover:bg-muted"
                      onClick={() => setQuery(historyQuery)}
                    >
                      <Clock className="h-3 w-3 mr-1" />
                      {historyQuery}
                    </Badge>
                  ))}
                </div>
              </div>
            )}
          </CardContent>
        </Card>

        {/* Rest of the component remains the same... */}
        {/* Search Results */}
        {hasSearched && (
          <div className="flex-1 space-y-4">
            {/* Results Header */}
            <div className="flex items-center justify-between">
              <div className="flex items-center space-x-4">
                <span className="text-sm text-muted-foreground">
                  {results.length} results found
                  {searchTime > 0 && ` in ${searchTime}ms`}
                </span>
              </div>

              <div className="flex items-center space-x-2">
                <Select value={sortBy} onValueChange={(value: any) => setSortBy(value)}>
                  <SelectTrigger className="w-36">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="relevance">Relevance</SelectItem>
                    <SelectItem value="date">Date modified</SelectItem>
                    <SelectItem value="name">Name</SelectItem>
                  </SelectContent>
                </Select>

                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setSortOrder(sortOrder === "asc" ? "desc" : "asc")}
                >
                  {sortOrder === "asc" ? (
                    <SortAsc className="h-4 w-4" />
                  ) : (
                    <SortDesc className="h-4 w-4" />
                  )}
                </Button>
              </div>
            </div>

            {/* Results List */}
            <div className="space-y-3 overflow-auto">
              {sortedResults.length === 0 ? (
                <Card>
                  <CardContent className="p-6 text-center">
                    <AlertCircle className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
                    <h3 className="text-lg font-medium mb-2">No files found</h3>
                    <p className="text-muted-foreground">
                      Try describing your file differently or using broader terms.
                    </p>
                  </CardContent>
                </Card>
              ) : (
                sortedResults.map((result, index) => {
                  const FileIcon = fileIconMap[result.file.extension] || FileText;

                  return (
                    <Card key={index} className="hover:shadow-md transition-shadow cursor-pointer">
                      <CardContent className="p-4">
                        <div className="flex items-start space-x-4">
                          <div className="flex-shrink-0">
                            <FileIcon className="h-10 w-10 text-primary" />
                          </div>

                          <div className="flex-1 min-w-0">
                            <div className="flex items-center justify-between mb-2">
                              <h3 className="text-lg font-medium truncate">
                                {highlightMatch(result.file.name, query)}
                              </h3>
                              <div className="flex items-center space-x-2">
                                <Badge variant="outline" className="flex items-center space-x-1">
                                  {getMatchTypeIcon(result.matchType)}
                                  <span>{getMatchTypeLabel(result.matchType)}</span>
                                </Badge>
                                <Badge variant="secondary">
                                  {Math.round(result.score * 100)}% match
                                </Badge>
                              </div>
                            </div>

                            <p className="text-sm text-muted-foreground mb-3">
                              {highlightMatch(result.excerpt, query)}
                            </p>

                            <div className="flex items-center justify-between">
                              <div className="flex items-center space-x-4 text-xs text-muted-foreground">
                                <span>{result.file.size}</span>
                                <span>{result.file.lastModified}</span>
                                {result.file.isDistributed && (
                                  <Badge variant="outline" className="text-xs">
                                    Distributed
                                  </Badge>
                                )}
                              </div>

                              <div className="flex items-center space-x-2">
                                <Button
                                  variant="outline"
                                  size="sm"
                                  onClick={() => handleDownload(result.file)}
                                >
                                  <Download className="h-4 w-4 mr-1" />
                                  Download
                                </Button>
                                {onFileSelect && (
                                  <Button
                                    variant="default"
                                    size="sm"
                                    onClick={() => onFileSelect(result.file)}
                                  >
                                    <ExternalLink className="h-4 w-4 mr-1" />
                                    Open
                                  </Button>
                                )}
                              </div>
                            </div>
                          </div>
                        </div>
                      </CardContent>
                    </Card>
                  );
                })
              )}
            </div>
          </div>
        )}

        {/* Empty State */}
        {!hasSearched && (
          <Card className="flex-1">
            <CardContent className="p-12 text-center">
              <Brain className="h-16 w-16 text-muted-foreground mx-auto mb-6" />
              <h2 className="text-2xl font-semibold mb-4">Intelligent File Search</h2>
              <p className="text-muted-foreground mb-6 max-w-md mx-auto">
                Describe what you're looking for in natural language. Our AI will analyze your files
                and find the most relevant matches based on content, context, and keywords.
              </p>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4 max-w-2xl mx-auto text-sm">
                <div className="flex items-center space-x-2 text-left">
                  <Sparkles className="h-4 w-4 text-primary" />
                  <span>Natural language search</span>
                </div>
                <div className="flex items-center space-x-2 text-left">
                  <FileText className="h-4 w-4 text-primary" />
                  <span>Content-based matching</span>
                </div>
                <div className="flex items-center space-x-2 text-left">
                  <Brain className="h-4 w-4 text-primary" />
                  <span>AI-powered relevance</span>
                </div>
                <div className="flex items-center space-x-2 text-left">
                  <Zap className="h-4 w-4 text-primary" />
                  <span>Fast semantic search</span>
                </div>
              </div>
            </CardContent>
          </Card>
        )}
      </div>
    </AppLayout>
  );
}
