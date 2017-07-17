#include <map>
#include <set>
#include <list>
#include <regex>
#include <fstream>
#include <iostream>
#include <cxxabi.h>

#include <rapidjson/writer.h>
#include <rapidjson/stringbuffer.h>

namespace
{

struct Deleter
{
    void operator()(char *data) const
    {
        free((void *) data);
    }
};

using CharPtr = std::unique_ptr<char, Deleter>;
}

std::string demangle(const std::string &mangledName)
{
    int status = 0;

    int shift = 0;
    if (mangledName[1] == '_')
    {
        shift = 1;
    }

    CharPtr realname(abi::__cxa_demangle(mangledName.data() + shift, 0, 0, &status));

    if (status == 0)
    {
        return std::string(realname.get());
    } else
    {
        if (mangledName[0] == '_')
        {
            const auto str = mangledName.substr(1, mangledName.size() - 1);
            int status = 0;
            CharPtr realname(abi::__cxa_demangle(str.data(), 0, 0, &status));
            if (status == 0)
            {
                return std::string(realname.get());
            }

            return mangledName;
        }

        return mangledName;
    }
}

void printUsage()
{
    std::cout << "webstackwalker crash_dump symbol_file" << std::endl;
}

std::map<std::string, std::string> SYMBOL_MAP;

void readSymbols(const std::string &path);

void flushUnParseLine(const std::string &line);

int main(int argc, char **argv)
{
    if (argc < 2)
    {
        printUsage();
        return 1;
    }

    readSymbols(std::string(argv[2]));

    const std::string inputFile(argv[1]);
    std::ifstream input(inputFile);
    const std::regex re("^(?:\\s{4}at\\s){0,1}(?:Array\\.){0,1}([\\w\\d\\$]+)(?: \\(|@).+\\){0,1}");

    rapidjson::StringBuffer buffer;
    rapidjson::Writer<rapidjson::StringBuffer> writer(buffer);
    writer.StartArray();

    const std::set<std::string> skip = {"jsStackTrace", "stackTrace", "abort"};

    while (!input.eof())
    {
        std::smatch match;
        std::string line;
        std::getline(input, line);

        if (std::regex_search(line, match, re))
        {
            if (skip.count(match[1]))
            {
                continue;
            }

            auto iter = SYMBOL_MAP.find(match[1]);
            std::string function;
            if (iter != SYMBOL_MAP.cend())
            {
                function = demangle(iter->second);
            } else
            {
                function = demangle(match[1]);
            }

            writer.String(function.c_str());
        }
    }
    writer.EndArray();

    std::cout << buffer.GetString() << std::endl;


    return 0;
}

void readSymbols(const std::string &path)
{
    std::ifstream input(path);

    if (!input.is_open())
    {
        std::cerr << "Can't open symbols file: " << path << std::endl;
        exit(2);
    }

    const std::regex re("^([\\d\\w$]+):([\\d\\w]+)$");

    while (!input.eof())
    {
        std::smatch match;
        std::string line;
        std::getline(input, line);
        if (std::regex_search(line, match, re))
        {
            SYMBOL_MAP[match[1]] = match[2];
        } else
        {
            flushUnParseLine(line);
        }
    }
}

void flushUnParseLine(const std::string &line)
{
    if (!line.empty())
    {
        std::ofstream ofs;
        ofs.open("/tmp/flushUnParseLines.txt", std::ofstream::out | std::ofstream::app);
        ofs << line << "\n";
        ofs.flush();
        ofs.close();
    }
}